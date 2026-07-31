package resource

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const identityTlsCertAuthMethodName = "tls-cert-auth"

func NewIdentityTlsCertAuthResource() resource.Resource {
	return &IdentityTlsCertAuthResource{}
}

type IdentityTlsCertAuthResource struct {
	client *infisical.Client
}

type IdentityTlsCertAuthResourceModel struct {
	ID                           types.String `tfsdk:"id"`
	IdentityID                   types.String `tfsdk:"identity_id"`
	CaCertificate                types.String `tfsdk:"ca_certificate"`
	AllowedCommonNames           types.List   `tfsdk:"allowed_common_names"`
	AllowedSubjectAltNames       types.List   `tfsdk:"allowed_subject_alt_names"`
	VerifyClientCertificateChain types.Bool   `tfsdk:"verify_client_certificate_chain"`
	AccessTokenTrustedIps        types.List   `tfsdk:"access_token_trusted_ips"`
	AccessTokenTTL               types.Int64  `tfsdk:"access_token_ttl"`
	AccessTokenMaxTTL            types.Int64  `tfsdk:"access_token_max_ttl"`
	AccessTokenNumUsesLimit      types.Int64  `tfsdk:"access_token_num_uses_limit"`
}

type IdentityTlsCertAuthResourceTrustedIps struct {
	IpAddress types.String `tfsdk:"ip_address"`
}

func (r *IdentityTlsCertAuthResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_tls_cert_auth"
}

func (r *IdentityTlsCertAuthResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage identity tls certificate auth in Infisical.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the tls certificate auth",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"identity_id": schema.StringAttribute{
				Description:   "The ID of the identity to attach the configuration onto.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"ca_certificate": schema.StringAttribute{
				Description: "The PEM-encoded CA certificate that client certificates must be issued by to authenticate with Infisical.",
				Required:    true,
			},
			"allowed_common_names": schema.ListAttribute{
				ElementType: types.StringType,
				Description: "List of trusted common names that client certificates must have to authenticate with Infisical. When omitted, any common name is accepted.",
				Optional:    true,
			},
			"allowed_subject_alt_names": schema.ListAttribute{
				ElementType:         types.StringType,
				Description:         "List of trusted subject alternative names that client certificates must have to authenticate with Infisical. Non-DNS entries must be prefixed with their type. When omitted, any subject alternative name is accepted.",
				MarkdownDescription: "List of trusted subject alternative names that client certificates must have to authenticate with Infisical. Non-DNS entries must be prefixed with their type (e.g. `URI:spiffe://example.org/service`, `IP:10.0.0.1`, `EMAIL:svc@example.com`). When omitted, any subject alternative name is accepted.",
				Optional:            true,
			},
			"verify_client_certificate_chain": schema.BoolAttribute{
				Description: "Whether to build and verify the full certificate chain presented by the client up to the configured CA certificate, instead of requiring the client certificate to be signed directly by it. Default: false",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
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

func (r *IdentityTlsCertAuthResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func tlsCertAuthAllowedCommonNamesAsApiField(ctx context.Context, diagnostics diag.Diagnostics, planField types.List) *string {
	allowedCommonNames := infisicaltf.StringListToGoStringSlice(ctx, diagnostics, planField)
	if len(allowedCommonNames) == 0 {
		return nil
	}

	joined := strings.Join(allowedCommonNames, ",")
	return &joined
}

func tlsCertAuthAllowedSubjectAltNamesAsApiField(ctx context.Context, diagnostics diag.Diagnostics, planField types.List) []string {
	allowedSubjectAltNames := infisicaltf.StringListToGoStringSlice(ctx, diagnostics, planField)
	if len(allowedSubjectAltNames) == 0 {
		return nil
	}

	return allowedSubjectAltNames
}

func tlsCertAuthStringListFromApi(ctx context.Context, diagnose diag.Diagnostics, values []string, current types.List) types.List {
	if len(values) == 0 {
		if !current.IsNull() && !current.IsUnknown() && len(current.Elements()) == 0 {
			return current
		}
		return types.ListNull(types.StringType)
	}

	list, diags := types.ListValueFrom(ctx, types.StringType, values)
	diagnose.Append(diags...)
	if diagnose.HasError() {
		return types.ListNull(types.StringType)
	}

	return list
}

func updateTlsCertAuthTerraformStateFromApi(ctx context.Context, diagnose diag.Diagnostics, plan *IdentityTlsCertAuthResourceModel, newIdentityTlsCertAuth *infisical.IdentityTlsCertAuth) {
	plan.AccessTokenMaxTTL = types.Int64Value(newIdentityTlsCertAuth.AccessTokenMaxTTL)
	plan.AccessTokenTTL = types.Int64Value(newIdentityTlsCertAuth.AccessTokenTTL)
	plan.AccessTokenNumUsesLimit = types.Int64Value(newIdentityTlsCertAuth.AccessTokenNumUsesLimit)
	plan.VerifyClientCertificateChain = types.BoolValue(newIdentityTlsCertAuth.VerifyClientCertificateChain)

	planAccessTokenTrustedIps := make([]IdentityTlsCertAuthResourceTrustedIps, len(newIdentityTlsCertAuth.AccessTokenTrustedIPS))
	for i, el := range newIdentityTlsCertAuth.AccessTokenTrustedIPS {
		if el.Prefix != nil {
			planAccessTokenTrustedIps[i] = IdentityTlsCertAuthResourceTrustedIps{IpAddress: types.StringValue(
				el.IpAddress + "/" + strconv.Itoa(*el.Prefix),
			)}
		} else {
			planAccessTokenTrustedIps[i] = IdentityTlsCertAuthResourceTrustedIps{IpAddress: types.StringValue(
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
	plan.AccessTokenTrustedIps = stateAccessTokenTrustedIps

	var allowedCommonNames []string
	if newIdentityTlsCertAuth.AllowedCommonNames != nil && *newIdentityTlsCertAuth.AllowedCommonNames != "" {
		allowedCommonNames = strings.Split(*newIdentityTlsCertAuth.AllowedCommonNames, ",")
	}

	plan.AllowedCommonNames = tlsCertAuthStringListFromApi(ctx, diagnose, allowedCommonNames, plan.AllowedCommonNames)
	plan.AllowedSubjectAltNames = tlsCertAuthStringListFromApi(ctx, diagnose, newIdentityTlsCertAuth.AllowedSubjectAltNames, plan.AllowedSubjectAltNames)
	if diagnose.HasError() {
		return
	}

	if newIdentityTlsCertAuth.CaCertificate != "" {
		plan.CaCertificate = types.StringValue(infisicaltf.PreserveStringIfTrimmedEqual(newIdentityTlsCertAuth.CaCertificate, plan.CaCertificate))
	}
}

func (r *IdentityTlsCertAuthResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create identity tls certificate auth",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan IdentityTlsCertAuthResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	accessTokenTrustedIps := tfPlanExpandIpFieldAsApiField(ctx, resp.Diagnostics, plan.AccessTokenTrustedIps)
	allowedCommonNames := tlsCertAuthAllowedCommonNamesAsApiField(ctx, resp.Diagnostics, plan.AllowedCommonNames)
	allowedSubjectAltNames := tlsCertAuthAllowedSubjectAltNamesAsApiField(ctx, resp.Diagnostics, plan.AllowedSubjectAltNames)
	if resp.Diagnostics.HasError() {
		return
	}

	newIdentityTlsCertAuth, err := r.client.CreateIdentityTlsCertAuth(infisical.CreateIdentityTlsCertAuthRequest{
		IdentityID:                   plan.IdentityID.ValueString(),
		CaCertificate:                plan.CaCertificate.ValueString(),
		AllowedCommonNames:           allowedCommonNames,
		AllowedSubjectAltNames:       allowedSubjectAltNames,
		VerifyClientCertificateChain: plan.VerifyClientCertificateChain.ValueBoolPointer(),
		AccessTokenTrustedIPS:        accessTokenTrustedIps,
		AccessTokenTTL:               infisicaltf.Int64PtrIfKnown(plan.AccessTokenTTL),
		AccessTokenMaxTTL:            infisicaltf.Int64PtrIfKnown(plan.AccessTokenMaxTTL),
		AccessTokenNumUsesLimit:      infisicaltf.Int64PtrIfKnown(plan.AccessTokenNumUsesLimit),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating identity tls certificate auth",
			"Couldn't create identity tls certificate auth in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(newIdentityTlsCertAuth.ID)
	updateTlsCertAuthTerraformStateFromApi(ctx, resp.Diagnostics, &plan, &newIdentityTlsCertAuth)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *IdentityTlsCertAuthResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read identity tls certificate auth",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state IdentityTlsCertAuthResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	identityTlsCertAuth, err := r.client.GetIdentityTlsCertAuth(infisical.GetIdentityTlsCertAuthRequest{
		IdentityID: state.IdentityID.ValueString(),
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError(
				"Error reading identity tls certificate auth",
				"Couldn't read identity tls certificate auth from Infisical, unexpected error: "+err.Error(),
			)
			return
		}
	}

	updateTlsCertAuthTerraformStateFromApi(ctx, resp.Diagnostics, &state, &identityTlsCertAuth)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *IdentityTlsCertAuthResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update identity tls certificate auth",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan IdentityTlsCertAuthResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state IdentityTlsCertAuthResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	accessTokenTrustedIps := tfPlanExpandIpFieldAsApiField(ctx, resp.Diagnostics, plan.AccessTokenTrustedIps)
	allowedCommonNames := tlsCertAuthAllowedCommonNamesAsApiField(ctx, resp.Diagnostics, plan.AllowedCommonNames)
	allowedSubjectAltNames := tlsCertAuthAllowedSubjectAltNamesAsApiField(ctx, resp.Diagnostics, plan.AllowedSubjectAltNames)
	if resp.Diagnostics.HasError() {
		return
	}

	updatedIdentityTlsCertAuth, err := r.client.UpdateIdentityTlsCertAuth(infisical.UpdateIdentityTlsCertAuthRequest{
		IdentityID:                   plan.IdentityID.ValueString(),
		CaCertificate:                plan.CaCertificate.ValueString(),
		AllowedCommonNames:           allowedCommonNames,
		AllowedSubjectAltNames:       allowedSubjectAltNames,
		VerifyClientCertificateChain: plan.VerifyClientCertificateChain.ValueBoolPointer(),
		AccessTokenTrustedIPS:        accessTokenTrustedIps,
		AccessTokenTTL:               infisicaltf.Int64PtrIfKnown(plan.AccessTokenTTL),
		AccessTokenMaxTTL:            infisicaltf.Int64PtrIfKnown(plan.AccessTokenMaxTTL),
		AccessTokenNumUsesLimit:      infisicaltf.Int64PtrIfKnown(plan.AccessTokenNumUsesLimit),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating identity tls certificate auth",
			"Couldn't update identity tls certificate auth in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	updateTlsCertAuthTerraformStateFromApi(ctx, resp.Diagnostics, &plan, &updatedIdentityTlsCertAuth)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *IdentityTlsCertAuthResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete identity tls certificate auth",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state IdentityTlsCertAuthResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.RevokeIdentityTlsCertAuth(infisical.RevokeIdentityTlsCertAuthRequest{
		IdentityID: state.IdentityID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting identity tls certificate auth",
			"Couldn't delete identity tls certificate auth from Infisical, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *IdentityTlsCertAuthResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to import identity tls certificate auth",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	identity, err := r.client.GetIdentity(infisical.GetIdentityRequest{
		IdentityID: req.ID,
	})
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.Diagnostics.AddError(
				"Identity not found",
				"The identity with the given ID was not found",
			)
		} else {
			resp.Diagnostics.AddError(
				"Error importing identity tls certificate auth",
				"Couldn't read identity from Infisical, unexpected error: "+err.Error(),
			)
		}
		return
	}

	if len(identity.Identity.AuthMethods) == 0 {
		resp.Diagnostics.AddError(
			"Identity tls certificate auth not found",
			"The identity with the given ID has no configured auth methods",
		)
		return
	}

	if !slices.Contains(identity.Identity.AuthMethods, identityTlsCertAuthMethodName) {
		resp.Diagnostics.AddError(
			"Identity tls certificate auth not found",
			"The identity with the given ID does not have tls certificate auth configured",
		)
		return
	}

	identityTlsCertAuth, err := r.client.GetIdentityTlsCertAuth(infisical.GetIdentityTlsCertAuthRequest{
		IdentityID: req.ID,
	})
	if err != nil {
		if err == infisical.ErrNotFound {
			resp.Diagnostics.AddError(
				"Identity tls certificate auth not found",
				"The identity with the given ID does not have tls certificate auth configured",
			)
		} else {
			resp.Diagnostics.AddError(
				"Error importing identity tls certificate auth",
				"Couldn't read identity tls certificate auth from Infisical, unexpected error: "+err.Error(),
			)
		}
		return
	}

	var state IdentityTlsCertAuthResourceModel

	state.ID = types.StringValue(identityTlsCertAuth.ID)
	state.IdentityID = types.StringValue(identityTlsCertAuth.IdentityID)
	updateTlsCertAuthTerraformStateFromApi(ctx, resp.Diagnostics, &state, &identityTlsCertAuth)
	if resp.Diagnostics.HasError() {
		return
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
