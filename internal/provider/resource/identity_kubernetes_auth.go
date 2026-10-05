package resource

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
	customtypes "terraform-provider-infisical/internal/pkg/customtypes"
	infisicalstrings "terraform-provider-infisical/internal/pkg/strings"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NewIdentityKubernetesAuthResource is a helper function to simplify the provider implementation.
func NewIdentityKubernetesAuthResource() resource.Resource {
	return &IdentityKubernetesAuthResource{}
}

// IdentityKubernetesAuthResource is the resource implementation.
type IdentityKubernetesAuthResource struct {
	client *infisical.Client
}

// IdentityKubernetesAuthResourceSourceModel describes the data source data model.
type IdentityKubernetesAuthResourceModel struct {
	ID                         types.String                   `tfsdk:"id"`
	IdentityID                 types.String                   `tfsdk:"identity_id"`
	KubernetesHost             types.String                   `tfsdk:"kubernetes_host"`
	CaCertificate              customtypes.TrimmedStringValue `tfsdk:"kubernetes_ca_certificate"`
	TokenReviewerJWT           types.String                   `tfsdk:"token_reviewer_jwt"`
	AllowedServiceAccountNames types.List                     `tfsdk:"allowed_service_account_names"`
	AllowedAudience            types.String                   `tfsdk:"allowed_audience"`
	AllowedNamespaces          types.List                     `tfsdk:"allowed_namespaces"`
	AccessTokenTTL             types.Int64                    `tfsdk:"access_token_ttl"`
	AccessTokenMaxTTL          types.Int64                    `tfsdk:"access_token_max_ttl"`
	AccessTokenNumUsesLimit    types.Int64                    `tfsdk:"access_token_num_uses_limit"`
	AccessTokenTrustedIps      types.List                     `tfsdk:"access_token_trusted_ips"`

	GatewayID         types.String `tfsdk:"gateway_id"`
	TokenReviewerMode types.String `tfsdk:"token_reviewer_mode"` // api|gateway (default is api)
	TemplateID        types.String `tfsdk:"template_id"`

	HasTemplateSourcedTokenReviewerJWT types.Bool `tfsdk:"has_template_sourced_token_reviewer_jwt"`
}

type IdentityKubernetesAuthResourceTrustedIps struct {
	IpAddress types.String `tfsdk:"ip_address"`
}

// Metadata returns the resource type name.
func (r *IdentityKubernetesAuthResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_kubernetes_auth"
}

const (
	TOKEN_REVIEWER_MODE_API     = "api"
	TOKEN_REVIEWER_MODE_GATEWAY = "gateway"
)

// Schema defines the schema for the resource.
func (r *IdentityKubernetesAuthResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage identity kubernetes auth in Infisical.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the kubernetes auth",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"identity_id": schema.StringAttribute{
				Description:   "The ID of the identity to attach the configuration onto.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			// Computed so a linked template can supply it, and read back while linked so that
			// removing template_id can tell a kept host from a changed one.
			"kubernetes_host": schema.StringAttribute{
				Description:   "The host string, host:port pair, or URL to the base of the Kubernetes API server. This can usually be obtained by running `kubectl cluster-info`.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{nullUnlessTemplateLinked{}},
			},
			"token_reviewer_jwt": schema.StringAttribute{
				Description:         "A long-lived service account JWT token for Infisical to access the TokenReview API to validate other service account JWT tokens submitted by applications/pods. This is the JWT token obtained from step 1.5.",
				MarkdownDescription: "A long-lived service account JWT token for Infisical to access the [TokenReview API](https://kubernetes.io/docs/reference/kubernetes-api/authentication-resources/token-review-v1/) to validate other service account JWT tokens submitted by applications/pods. This is the JWT token obtained from step 1.5.",
				Optional:            true,
			},

			// Computed so a linked template can supply it.
			"gateway_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Select a gateway for private cluster access. If not specified, the Internet Gateway will be used.",
				PlanModifiers: []planmodifier.String{nullUnlessTemplateLinked{}},
			},
			"token_reviewer_mode": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Choose between Token ('api') or 'gateway' authentication. If using Gateway, the Gateway must be deployed in your Kubernetes cluster. Defaults to `api`, or to the linked template's mode when `template_id` is set.",
				Validators: []validator.String{
					stringvalidator.OneOf(TOKEN_REVIEWER_MODE_API, TOKEN_REVIEWER_MODE_GATEWAY),
				},
				// A static default would fight the mode a linked template supplies.
				PlanModifiers: []planmodifier.String{defaultUnlessTemplateLinked{value: TOKEN_REVIEWER_MODE_API}},
			},
			"template_id": schema.StringAttribute{
				Optional:    true,
				Description: "The ID of an `infisical_identity_kubernetes_auth_template` to take the Kubernetes connection settings from. When set, `kubernetes_host`, `kubernetes_ca_certificate`, `token_reviewer_jwt`, `token_reviewer_mode`, `gateway_id` and `allowed_audience` come from the template and must not be set here, and later edits to the template propagate to this identity. Removing it unlinks the template and applies the settings in this configuration instead: a connection setting the configuration leaves out is cleared, and the plan shows it. To unlink and keep the template's settings, set them here to the template's values (for example from the `infisical_identity_kubernetes_auth_template` data source); an unlink that changes nothing else then only needs the `unlink-templates` permission on auth templates, or `edit-auth` on the identity. A token reviewer JWT copied from the template is kept unless `token_reviewer_jwt` is set to a new value, or to `\"\"` to remove it.",
				Validators: []validator.String{
					infisicaltf.UuidValidator,
					stringvalidator.ConflictsWith(
						path.MatchRoot("kubernetes_host"),
						path.MatchRoot("kubernetes_ca_certificate"),
						path.MatchRoot("token_reviewer_jwt"),
						path.MatchRoot("token_reviewer_mode"),
						path.MatchRoot("gateway_id"),
						path.MatchRoot("allowed_audience"),
					),
				},
			},

			"kubernetes_ca_certificate": schema.StringAttribute{
				CustomType:          customtypes.TrimmedStringType{},
				Description:         "The PEM-encoded CA cert for the Kubernetes API server. This is used by the TLS client for secure communication with the Kubernetes API server.",
				MarkdownDescription: "The PEM-encoded CA cert for the Kubernetes API server. This is used by the TLS client for secure communication with the Kubernetes API server.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{clearWhenUnlinking{}},
			},
			"has_template_sourced_token_reviewer_jwt": schema.BoolAttribute{
				Description: "Whether this identity holds a token reviewer JWT copied from an auth template. That JWT is write-only and reads back as empty, so this is the only sign it is there. It is kept after `template_id` is removed unless `token_reviewer_jwt` replaces it, or is set to `\"\"` to remove it.",
				Computed:    true,
			},
			"allowed_service_account_names": schema.ListAttribute{
				ElementType: types.StringType,
				Description: "List of trusted service account names that are allowed to authenticate with Infisical.",
				Optional:    true,
				Computed:    true,
			},
			"allowed_audience": schema.StringAttribute{
				Description:   "An optional audience claim that the service account JWT token must have to authenticate with Infisical.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{clearWhenUnlinking{}},
			},
			"allowed_namespaces": schema.ListAttribute{
				ElementType: types.StringType,
				Description: "List of trusted namespaces that service accounts must belong to authenticate with Infisical.",
				Optional:    true,
				Computed:    true,
			},
			"access_token_trusted_ips": schema.ListNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "A list of IPs or CIDR ranges that access tokens can be used from. You can use 0.0.0.0/0, to allow usage from any network address..",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ip_address": schema.StringAttribute{
							Optional: true,
							Computed: true,
						},
					},
				},
			},
			"access_token_ttl": schema.Int64Attribute{
				Description: "The lifetime for an access token in seconds. This value will be referenced at renewal time. Default: 2592000",
				Computed:    true,
				Optional:    true,
			},
			"access_token_max_ttl": schema.Int64Attribute{
				Description: "The maximum lifetime for an access token in seconds. This value will be referenced at renewal time. Default: 2592000",
				Computed:    true,
				Optional:    true,
			},
			"access_token_num_uses_limit": schema.Int64Attribute{
				Description: "The maximum number of times that an access token can be used; a value of 0 implies infinite number of uses. Default:0",
				Computed:    true,
				Optional:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *IdentityKubernetesAuthResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*infisical.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected resource Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func updateKubernetesAuthStateByApi(ctx context.Context, diagnose diag.Diagnostics, plan *IdentityKubernetesAuthResourceModel, newIdentityKubernetesAuth *infisical.IdentityKubernetesAuth) {
	plan.AccessTokenMaxTTL = types.Int64Value(newIdentityKubernetesAuth.AccessTokenMaxTTL)
	plan.AccessTokenTTL = types.Int64Value(newIdentityKubernetesAuth.AccessTokenTTL)
	plan.AccessTokenNumUsesLimit = types.Int64Value(newIdentityKubernetesAuth.AccessTokenNumUsesLimit)
	plan.AllowedAudience = types.StringValue(newIdentityKubernetesAuth.AllowedAudience)
	plan.CaCertificate = customtypes.NewTrimmedStringValue(newIdentityKubernetesAuth.CACERT)
	plan.TokenReviewerMode = types.StringValue(newIdentityKubernetesAuth.TokenReviewerMode)

	if newIdentityKubernetesAuth.GatewayID != "" {
		plan.GatewayID = types.StringValue(newIdentityKubernetesAuth.GatewayID)
	} else if plan.GatewayID.IsUnknown() {
		plan.GatewayID = types.StringNull()
	}

	if newIdentityKubernetesAuth.TemplateID != nil && *newIdentityKubernetesAuth.TemplateID != "" {
		plan.TemplateID = types.StringValue(*newIdentityKubernetesAuth.TemplateID)
		// The template owns the host while linked, so state follows the API. Unlinked, the
		// configured spelling is kept, as it always was.
		plan.KubernetesHost = types.StringNull()
		if newIdentityKubernetesAuth.KubernetesHost != "" {
			plan.KubernetesHost = types.StringValue(newIdentityKubernetesAuth.KubernetesHost)
		}
	} else {
		plan.TemplateID = types.StringNull()
		if plan.KubernetesHost.IsUnknown() {
			plan.KubernetesHost = types.StringNull()
			if newIdentityKubernetesAuth.KubernetesHost != "" {
				plan.KubernetesHost = types.StringValue(newIdentityKubernetesAuth.KubernetesHost)
			}
		}
	}
	plan.HasTemplateSourcedTokenReviewerJWT = types.BoolValue(newIdentityKubernetesAuth.IsTokenReviewerJwtTemplateSourced)

	planAccessTokenTrustedIps := make([]IdentityKubernetesAuthResourceTrustedIps, len(newIdentityKubernetesAuth.AccessTokenTrustedIPS))
	for i, el := range newIdentityKubernetesAuth.AccessTokenTrustedIPS {
		if el.Prefix != nil {
			planAccessTokenTrustedIps[i] = IdentityKubernetesAuthResourceTrustedIps{IpAddress: types.StringValue(
				el.IpAddress + "/" + strconv.Itoa(*el.Prefix),
			)}
		} else {
			planAccessTokenTrustedIps[i] = IdentityKubernetesAuthResourceTrustedIps{IpAddress: types.StringValue(
				el.IpAddress,
			)}
		}
	}

	stateAccessTokenTrustedIps, diags := types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"ip_address": types.StringType,
		},
	}, planAccessTokenTrustedIps)
	diagnose.Append(diags...)
	if diagnose.HasError() {
		return
	}

	plan.AllowedNamespaces, diags = types.ListValueFrom(ctx, types.StringType, infisicalstrings.StringSplitAndTrim(newIdentityKubernetesAuth.AllowedNamespaces, ","))
	diagnose.Append(diags...)
	if diagnose.HasError() {
		return
	}

	plan.AllowedServiceAccountNames, diags = types.ListValueFrom(ctx, types.StringType, infisicalstrings.StringSplitAndTrim(newIdentityKubernetesAuth.AllowedServiceAccountNames, ","))
	diagnose.Append(diags...)
	if diagnose.HasError() {
		return
	}

	plan.AccessTokenTrustedIps = stateAccessTokenTrustedIps
}

func validateTokenReviewerMode(diagnose *diag.Diagnostics, plan *IdentityKubernetesAuthResourceModel) {

	if plan == nil {
		diagnose.AddError(
			"Invalid plan",
			"Plan is nil",
		)
		return
	}

	tokenReviewerMode := plan.TokenReviewerMode.ValueString()

	if tokenReviewerMode == TOKEN_REVIEWER_MODE_GATEWAY {
		if plan.GatewayID.ValueString() == "" {
			diagnose.AddError(
				"Gateway ID is required",
				"Gateway ID is required when token reviewer mode is gateway. Please provide a valid gateway ID.",
			)
			return
		}

		if strings.TrimSpace(plan.CaCertificate.ValueString()) != "" {
			diagnose.AddError(
				"Cannot set CA certificate",
				"CA certificate is not allowed when token reviewer mode is gateway. Please remove the CA certificate.",
			)
			return
		}

		if strings.TrimSpace(plan.KubernetesHost.ValueString()) != "" {
			diagnose.AddError(
				"Cannot set Kubernetes host",
				"Kubernetes host is not allowed when token reviewer mode is gateway. Please remove the Kubernetes host.",
			)
			return
		}

		if strings.TrimSpace(plan.TokenReviewerJWT.ValueString()) != "" {
			diagnose.AddError(
				"Cannot set Token reviewer JWT",
				"Token reviewer JWT is not allowed when token reviewer mode is gateway. Please remove the Token reviewer JWT.",
			)
			return
		}
	} else if tokenReviewerMode == TOKEN_REVIEWER_MODE_API {
		// plan.TokenReviewerJWT is optional. if set to nothing, the auth method will act as self-reviewer.

		if strings.TrimSpace(plan.KubernetesHost.ValueString()) == "" {
			diagnose.AddError(
				"Kubernetes host is required",
				"Kubernetes host is required when token reviewer mode is api. Please provide a valid Kubernetes host.",
			)
			return
		}
	} else {
		diagnose.AddError(
			"Invalid token reviewer mode",
			"Invalid token reviewer mode. Please provide a valid token reviewer mode. Must be one of: "+TOKEN_REVIEWER_MODE_API+" or "+TOKEN_REVIEWER_MODE_GATEWAY,
		)
		return
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *IdentityKubernetesAuthResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create identity kubernetes auth",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan IdentityKubernetesAuthResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	accessTokenTrustedIps := tfPlanExpandIpFieldAsApiField(ctx, resp.Diagnostics, plan.AccessTokenTrustedIps)
	allowedNamespacpes := infisicaltf.StringListToGoStringSlice(ctx, resp.Diagnostics, plan.AllowedNamespaces)
	allowedNames := infisicaltf.StringListToGoStringSlice(ctx, resp.Diagnostics, plan.AllowedServiceAccountNames)

	var newIdentityKubernetesAuth infisical.IdentityKubernetesAuth
	var err error
	if !plan.TemplateID.IsNull() {
		newIdentityKubernetesAuth, err = r.client.CreateIdentityKubernetesAuthFromTemplate(infisical.CreateIdentityKubernetesAuthFromTemplateRequest{
			IdentityID:              plan.IdentityID.ValueString(),
			TemplateID:              plan.TemplateID.ValueString(),
			AccessTokenTTL:          plan.AccessTokenTTL.ValueInt64(),
			AccessTokenMaxTTL:       plan.AccessTokenMaxTTL.ValueInt64(),
			AccessTokenNumUsesLimit: plan.AccessTokenNumUsesLimit.ValueInt64(),
			AccessTokenTrustedIPS:   accessTokenTrustedIps,
			AllowedNamespaces:       strings.Join(allowedNamespacpes, ","),
			AllowedNames:            strings.Join(allowedNames, ","),
		})
	} else {
		newIdentityKubernetesAuth, err = r.createCustomKubernetesAuth(&resp.Diagnostics, &plan, accessTokenTrustedIps, allowedNamespacpes, allowedNames)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating identity kubernetes auth",
			"Couldn't save kubernetes auth to Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(newIdentityKubernetesAuth.ID)
	updateKubernetesAuthStateByApi(ctx, resp.Diagnostics, &plan, &newIdentityKubernetesAuth)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

func (r *IdentityKubernetesAuthResource) createCustomKubernetesAuth(diagnostics *diag.Diagnostics, plan *IdentityKubernetesAuthResourceModel, accessTokenTrustedIps []infisical.IdentityAuthTrustedIpRequest, allowedNamespacpes []string, allowedNames []string) (infisical.IdentityKubernetesAuth, error) {
	validateTokenReviewerMode(diagnostics, plan)
	if diagnostics.HasError() {
		return infisical.IdentityKubernetesAuth{}, nil
	}

	var kubernetesHost *string = nil
	if !plan.KubernetesHost.IsNull() && !plan.KubernetesHost.IsUnknown() && plan.KubernetesHost.ValueString() != "" {
		host := plan.KubernetesHost.ValueString()
		kubernetesHost = &host
	}

	var gatewayID *string = nil
	if !plan.GatewayID.IsNull() && !plan.GatewayID.IsUnknown() && plan.GatewayID.ValueString() != "" {
		gId := plan.GatewayID.ValueString()
		gatewayID = &gId
	}

	return r.client.CreateIdentityKubernetesAuth(infisical.CreateIdentityKubernetesAuthRequest{
		IdentityID:              plan.IdentityID.ValueString(),
		TokenReviewerMode:       plan.TokenReviewerMode.ValueString(),
		GatewayID:               gatewayID,
		AccessTokenTTL:          plan.AccessTokenTTL.ValueInt64(),
		AccessTokenMaxTTL:       plan.AccessTokenMaxTTL.ValueInt64(),
		AccessTokenNumUsesLimit: plan.AccessTokenNumUsesLimit.ValueInt64(),
		AccessTokenTrustedIPS:   accessTokenTrustedIps,
		KubernetesHost:          kubernetesHost,
		CACERT:                  plan.CaCertificate.ValueString(),
		TokenReviewerJwt:        plan.TokenReviewerJWT.ValueString(),
		AllowedNamespaces:       strings.Join(allowedNamespacpes, ","),
		AllowedNames:            strings.Join(allowedNames, ","),
		AllowedAudience:         plan.AllowedAudience.ValueString(),
	})
}

// Read refreshes the Terraform state with the latest data.
func (r *IdentityKubernetesAuthResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read identity kubernetes auth role",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state IdentityKubernetesAuthResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the latest data from the API
	identityKubernetesAuth, err := r.client.GetIdentityKubernetesAuth(infisical.GetIdentityKubernetesAuthRequest{
		IdentityID: state.IdentityID.ValueString(),
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError(
				"Error reading identity kubernetes auth",
				"Couldn't read identity kubernetes auth from Infisical, unexpected error: "+err.Error(),
			)
			return
		}
	}

	updateKubernetesAuthStateByApi(ctx, resp.Diagnostics, &state, &identityKubernetesAuth)
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *IdentityKubernetesAuthResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update identity kubernetes auth",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan IdentityKubernetesAuthResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state IdentityKubernetesAuthResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	accessTokenTrustedIps := tfPlanExpandIpFieldAsApiField(ctx, resp.Diagnostics, plan.AccessTokenTrustedIps)
	allowedNamespacpes := infisicaltf.StringListToGoStringSlice(ctx, resp.Diagnostics, plan.AllowedNamespaces)
	allowedNames := infisicaltf.StringListToGoStringSlice(ctx, resp.Diagnostics, plan.AllowedServiceAccountNames)

	var updatedIdentityKubernetesAuth infisical.IdentityKubernetesAuth
	var err error
	if kubernetesAuthUnlinksOnly(plan, state) {
		updatedIdentityKubernetesAuth, err = r.unlinkTemplateOnly(state)
	} else if !plan.TemplateID.IsNull() {
		updatedIdentityKubernetesAuth, err = r.client.UpdateIdentityKubernetesAuthFromTemplate(infisical.UpdateIdentityKubernetesAuthFromTemplateRequest{
			IdentityID:              plan.IdentityID.ValueString(),
			TemplateID:              plan.TemplateID.ValueString(),
			AccessTokenTrustedIPS:   accessTokenTrustedIps,
			AccessTokenTTL:          plan.AccessTokenTTL.ValueInt64(),
			AccessTokenMaxTTL:       plan.AccessTokenMaxTTL.ValueInt64(),
			AccessTokenNumUsesLimit: plan.AccessTokenNumUsesLimit.ValueInt64(),
			AllowedNamespaces:       strings.Join(allowedNamespacpes, ","),
			AllowedNames:            strings.Join(allowedNames, ","),
		})
	} else {
		// An unset JWT keeps one copied from a template: it reads back
		// as empty, so leaving it out of the configuration cannot mean "remove it". "" removes it.
		keepTemplateJwt := plan.TokenReviewerJWT.IsNull() && state.HasTemplateSourcedTokenReviewerJWT.ValueBool()
		updatedIdentityKubernetesAuth, err = r.updateCustomKubernetesAuth(&resp.Diagnostics, &plan, accessTokenTrustedIps, allowedNamespacpes, allowedNames, keepTemplateJwt)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating identity kubernetes auth",
			"Couldn't update identity kubernetes auth from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	updateKubernetesAuthStateByApi(ctx, resp.Diagnostics, &plan, &updatedIdentityKubernetesAuth)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *IdentityKubernetesAuthResource) updateCustomKubernetesAuth(diagnostics *diag.Diagnostics, plan *IdentityKubernetesAuthResourceModel, accessTokenTrustedIps []infisical.IdentityAuthTrustedIpRequest, allowedNamespacpes []string, allowedNames []string, keepTokenReviewerJwt bool) (infisical.IdentityKubernetesAuth, error) {
	validateTokenReviewerMode(diagnostics, plan)
	if diagnostics.HasError() {
		return infisical.IdentityKubernetesAuth{}, nil
	}

	var kubernetesHost *string = nil
	if !plan.KubernetesHost.IsNull() && !plan.KubernetesHost.IsUnknown() && plan.KubernetesHost.ValueString() != "" {
		host := plan.KubernetesHost.ValueString()
		kubernetesHost = &host
	}

	var gatewayID *string = nil
	if !plan.GatewayID.IsNull() && !plan.GatewayID.IsUnknown() && plan.GatewayID.ValueString() != "" {
		gId := plan.GatewayID.ValueString()
		gatewayID = &gId
	}

	var tokenReviewerJwt *string = nil
	if !plan.TokenReviewerJWT.IsNull() && !plan.TokenReviewerJWT.IsUnknown() && plan.TokenReviewerJWT.ValueString() != "" {
		reviewerJwt := plan.TokenReviewerJWT.ValueString()
		tokenReviewerJwt = &reviewerJwt
	}

	return r.client.UpdateIdentityKubernetesAuth(infisical.UpdateIdentityKubernetesAuthRequest{
		IdentityID:              plan.IdentityID.ValueString(),
		AccessTokenTrustedIPS:   accessTokenTrustedIps,
		TokenReviewerMode:       plan.TokenReviewerMode.ValueString(),
		GatewayID:               gatewayID,
		AccessTokenTTL:          plan.AccessTokenTTL.ValueInt64(),
		AccessTokenMaxTTL:       plan.AccessTokenMaxTTL.ValueInt64(),
		AccessTokenNumUsesLimit: plan.AccessTokenNumUsesLimit.ValueInt64(),
		KubernetesHost:          kubernetesHost,
		CACERT:                  plan.CaCertificate.ValueString(),
		TokenReviewerJwt:        tokenReviewerJwt,
		AllowedNamespaces:       strings.Join(allowedNamespacpes, ","),
		AllowedNames:            strings.Join(allowedNames, ","),
		AllowedAudience:         plan.AllowedAudience.ValueString(),
		// Null unlinks a template this identity was linked to.
		TemplateID:           nil,
		KeepTokenReviewerJwt: keepTokenReviewerJwt,
	})
}

// kubernetesAuthUnlinksOnly reports whether the plan removes template_id and changes nothing else,
// which the configuration expresses by setting the template's values (or leaving to the API what
// it keeps). Such a plan needs only the link removed, not an edit of the auth configuration.
func kubernetesAuthUnlinksOnly(plan, state IdentityKubernetesAuthResourceModel) bool {
	if state.TemplateID.IsNull() || !plan.TemplateID.IsNull() {
		return false
	}

	// A configured JWT replaces the stored one and "" removes it; both are edits.
	if !plan.TokenReviewerJWT.IsNull() {
		return false
	}

	return plan.KubernetesHost.Equal(state.KubernetesHost) &&
		!plan.CaCertificate.IsUnknown() &&
		strings.TrimSpace(plan.CaCertificate.ValueString()) == strings.TrimSpace(state.CaCertificate.ValueString()) &&
		plan.TokenReviewerMode.Equal(state.TokenReviewerMode) &&
		plan.GatewayID.Equal(state.GatewayID) &&
		plan.AllowedAudience.Equal(state.AllowedAudience) &&
		allowlistUnchanged(plan.AllowedNamespaces, state.AllowedNamespaces) &&
		allowlistUnchanged(plan.AllowedServiceAccountNames, state.AllowedServiceAccountNames) &&
		// Unset TTLs and trusted IPs are left out of an update, so the API keeps them.
		unchangedOrDefaulted(plan.AccessTokenTTL, state.AccessTokenTTL) &&
		unchangedOrDefaulted(plan.AccessTokenMaxTTL, state.AccessTokenMaxTTL) &&
		unchangedOrDefaulted(plan.AccessTokenNumUsesLimit, state.AccessTokenNumUsesLimit) &&
		unchangedOrDefaulted(plan.AccessTokenTrustedIps, state.AccessTokenTrustedIps)
}

// allowlistUnchanged is true when an update would leave an allowlist as it is. An unknown one, which
// is what an unset allowlist plans as on any change, is sent as empty: that only clears something
// when state holds entries.
func allowlistUnchanged(planned, stored types.List) bool {
	if planned.IsUnknown() {
		return !stored.IsUnknown() && len(stored.Elements()) == 0
	}
	return planned.Equal(stored)
}

// unlinkTemplateOnly removes the template link and nothing else. It goes through the template's
// delete-usage endpoint first, which needs only unlink-templates, so a role scoped to unlinking can
// do it. That endpoint also requires the template to exist and the org's plan to include
// templates, so on any failure it falls back to the identity's own update endpoint, which needs
// edit-auth and has neither requirement. Either way the identity keeps the settings copied onto it.
func (r *IdentityKubernetesAuthResource) unlinkTemplateOnly(state IdentityKubernetesAuthResourceModel) (infisical.IdentityKubernetesAuth, error) {
	identityID := state.IdentityID.ValueString()

	usageErr := r.client.UnlinkIdentityAuthTemplateUsage(state.TemplateID.ValueString(), []string{identityID})
	if usageErr != nil {
		if _, err := r.client.UnlinkIdentityKubernetesAuthTemplate(identityID); err != nil {
			return infisical.IdentityKubernetesAuth{}, fmt.Errorf(
				"unlinking needs unlink-templates on auth templates or edit-auth on the identity; the template's delete-usage endpoint failed (%w) and so did the identity's update endpoint (%w)",
				usageErr, err,
			)
		}
	}

	return r.client.GetIdentityKubernetesAuth(infisical.GetIdentityKubernetesAuthRequest{IdentityID: identityID})
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *IdentityKubernetesAuthResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete identity kubernetes auth",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state IdentityKubernetesAuthResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.RevokeIdentityKubernetesAuth(infisical.RevokeIdentityKubernetesAuthRequest{
		IdentityID: state.IdentityID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting identity kubernetes auth",
			"Couldn't delete identity kubernetes auth from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

}

func (r *IdentityKubernetesAuthResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to import identity kubernetes auth",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	resource.ImportStatePassthroughID(ctx, path.Root("identity_id"), req, resp)
}
