package resource

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
	customtypes "terraform-provider-infisical/internal/pkg/customtypes"
	pkg "terraform-provider-infisical/internal/pkg/modifiers"
	infisicalstrings "terraform-provider-infisical/internal/pkg/strings"
	"terraform-provider-infisical/internal/pkg/terraform"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NewIdentityOidcAuthResource is a helper function to simplify the provider implementation.
func NewIdentityOidcAuthResource() resource.Resource {
	return &IdentityOidcAuthResource{}
}

// IdentityOidcAuthResource is the resource implementation.
type IdentityOidcAuthResource struct {
	client *infisical.Client
}

// IdentityOidcAuthResourceSourceModel describes the data source data model.
type IdentityOidcAuthResourceModel struct {
	ID                      types.String                   `tfsdk:"id"`
	IdentityID              types.String                   `tfsdk:"identity_id"`
	OidcDiscoveryUrl        types.String                   `tfsdk:"oidc_discovery_url"`
	CaCertificate           customtypes.TrimmedStringValue `tfsdk:"oidc_ca_certificate"`
	BoundIssuer             types.String                   `tfsdk:"bound_issuer"`
	BoundAudiences          types.List                     `tfsdk:"bound_audiences"`
	BoundClaims             types.Map                      `tfsdk:"bound_claims"`
	ClaimMetadataMapping    types.Map                      `tfsdk:"claim_metadata_mapping"`
	BoundSubject            types.String                   `tfsdk:"bound_subject"`
	AccessTokenTrustedIps   types.List                     `tfsdk:"access_token_trusted_ips"`
	AccessTokenTTL          types.Int64                    `tfsdk:"access_token_ttl"`
	AccessTokenMaxTTL       types.Int64                    `tfsdk:"access_token_max_ttl"`
	AccessTokenNumUsesLimit types.Int64                    `tfsdk:"access_token_num_uses_limit"`
	TemplateID              types.String                   `tfsdk:"template_id"`
}

var (
	_ resource.ResourceWithValidateConfig = &IdentityOidcAuthResource{}
)

type IdentityOidcAuthResourceTrustedIps struct {
	IpAddress types.String `tfsdk:"ip_address"`
}

// Metadata returns the resource type name.
func (r *IdentityOidcAuthResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_oidc_auth"
}

// Schema defines the schema for the resource.
func (r *IdentityOidcAuthResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage identity oidc auth in Infisical.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the oidc auth.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"identity_id": schema.StringAttribute{
				Description:   "The ID of the identity to attach the configuration onto.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			// Optional and computed so a linked template can supply it; required otherwise, which
			// ValidateConfig enforces.
			"oidc_discovery_url": schema.StringAttribute{
				Description:   "The URL used to retrieve the OpenID Connect configuration from the identity provider. Required unless `template_id` is set.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{useStateForUnknownUnlessTemplateLinked{}},
			},
			"bound_issuer": schema.StringAttribute{
				Description:   "The unique identifier of the identity provider issuing the OIDC tokens. Required unless `template_id` is set.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{useStateForUnknownUnlessTemplateLinked{}},
			},
			"bound_audiences": schema.ListAttribute{
				Description:   "The comma-separated list of intended recipients.",
				Optional:      true,
				Computed:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.List{useStateForUnknownUnlessTemplateLinked{}, clearWhenUnlinking{}},
			},
			"bound_claims": schema.MapAttribute{
				Description: "The attributes that should be present in the JWT for it to be valid. The provided values can be a glob pattern.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
					pkg.CommaSpaceMapModifier{},
				},
			},

			"claim_metadata_mapping": schema.MapAttribute{
				Description: "Map OIDC token claims to metadata fields. Example: {\"role\": \"token.groups\"}, this would become identity.metadata.oidc.claims.role",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"bound_subject": schema.StringAttribute{
				Description:   "The expected principal that is the subject of the JWT.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"oidc_ca_certificate": schema.StringAttribute{
				CustomType:          customtypes.TrimmedStringType{},
				Description:         "The PEM-encoded CA cert for establishing secure communication with the Identity Provider endpoints",
				MarkdownDescription: "The PEM-encoded CA cert for establishing secure communication with the Identity Provider endpoints",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{useStateForUnknownUnlessTemplateLinked{}, clearWhenUnlinking{}},
			},
			"template_id": schema.StringAttribute{
				Optional:    true,
				Description: "The ID of an `infisical_identity_oidc_auth_template` to take the identity provider settings from. When set, `oidc_discovery_url`, `bound_issuer`, `bound_audiences` and `oidc_ca_certificate` come from the template and must not be set here, later edits to the template propagate to this identity, and `bound_subject` or `bound_claims` must restrict which workloads can authenticate. Removing it unlinks the template and applies the settings in this configuration instead: `bound_audiences` and `oidc_ca_certificate` are cleared when the configuration leaves them out, and the plan shows it. To unlink and keep the template's settings, set them here to the template's values (for example from the `infisical_identity_oidc_auth_template` data source); an unlink that changes nothing else then only needs the `unlink-templates` permission on auth templates, or `edit-auth` on the identity.",
				Validators: []validator.String{
					terraform.UuidValidator,
					stringvalidator.ConflictsWith(
						path.MatchRoot("oidc_discovery_url"),
						path.MatchRoot("bound_issuer"),
						path.MatchRoot("bound_audiences"),
						path.MatchRoot("oidc_ca_certificate"),
					),
				},
			},
			"access_token_trusted_ips": schema.ListNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "A list of IPs or CIDR ranges that access tokens can be used from. You can use 0.0.0.0/0, to allow usage from any network address...",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ip_address": schema.StringAttribute{
							Optional: true,
							Computed: true,
						},
					},
				},
				PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"access_token_ttl": schema.Int64Attribute{
				Description:   "The lifetime for an access token in seconds. This value will be referenced at renewal time. Default: 2592000",
				Computed:      true,
				Optional:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"access_token_max_ttl": schema.Int64Attribute{
				Description:   "The maximum lifetime for an access token in seconds. This value will be referenced at renewal time. Default: 2592000",
				Computed:      true,
				Optional:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"access_token_num_uses_limit": schema.Int64Attribute{
				Description:   "The maximum number of times that an access token can be used; a value of 0 implies infinite number of uses. Default:0",
				Computed:      true,
				Optional:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
		},
	}
}

// The provider settings are required unless a linked template supplies them, and a linked
// identity must carry its own principal binding, since the template pins only the issuer.
func (r *IdentityOidcAuthResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config IdentityOidcAuthResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.TemplateID.IsNull() {
		for attribute, value := range map[string]types.String{
			"oidc_discovery_url": config.OidcDiscoveryUrl,
			"bound_issuer":       config.BoundIssuer,
		} {
			if value.IsNull() {
				resp.Diagnostics.AddAttributeError(
					path.Root(attribute),
					"Missing required argument",
					attribute+" is required unless template_id is set.",
				)
			}
		}
		return
	}

	if config.BoundSubject.IsUnknown() || config.BoundClaims.IsUnknown() {
		return
	}
	if config.BoundSubject.ValueString() == "" && len(config.BoundClaims.Elements()) == 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root("template_id"),
			"No principal binding configured",
			"An identity linked to an auth template must set bound_subject or at least one bound_claims entry. The template pins only the issuer, so without a binding any token that issuer signs could authenticate as this identity.",
		)
	}
}

// Configure adds the provider configured client to the resource.
func (r *IdentityOidcAuthResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func updateOidcAuthStateByApi(ctx context.Context, diagnose diag.Diagnostics, plan *IdentityOidcAuthResourceModel, newIdentityOidcAuth *infisical.IdentityOidcAuth) {
	plan.AccessTokenMaxTTL = types.Int64Value(newIdentityOidcAuth.AccessTokenMaxTTL)
	plan.AccessTokenTTL = types.Int64Value(newIdentityOidcAuth.AccessTokenTTL)
	plan.AccessTokenNumUsesLimit = types.Int64Value(newIdentityOidcAuth.AccessTokenNumUsesLimit)

	plan.OidcDiscoveryUrl = types.StringValue(newIdentityOidcAuth.OidcDiscoveryUrl)
	plan.BoundIssuer = types.StringValue(newIdentityOidcAuth.BoundIssuer)
	plan.BoundSubject = types.StringValue(newIdentityOidcAuth.BoundSubject)
	plan.CaCertificate = customtypes.NewTrimmedStringValue(newIdentityOidcAuth.CACERT)

	if newIdentityOidcAuth.TemplateID != nil && *newIdentityOidcAuth.TemplateID != "" {
		plan.TemplateID = types.StringValue(*newIdentityOidcAuth.TemplateID)
	} else {
		plan.TemplateID = types.StringNull()
	}

	boundClaimsElements := make(map[string]attr.Value)
	claimMetadataMappingElements := make(map[string]attr.Value)
	for key, value := range newIdentityOidcAuth.BoundClaims {
		// Check plan format
		useSpaces := false
		if !plan.BoundClaims.IsNull() {
			if planValue, ok := plan.BoundClaims.Elements()[key]; ok {
				if planStr, ok := planValue.(types.String); ok {
					useSpaces = strings.Contains(planStr.ValueString(), ", ")
				}
			}
		}

		// Split and normalize
		parts := strings.Split(value, ",")
		for i, part := range parts {
			parts[i] = strings.TrimSpace(part)
		}

		// Use the same format as the plan
		if useSpaces {
			boundClaimsElements[key] = types.StringValue(strings.Join(parts, ", "))
		} else {
			boundClaimsElements[key] = types.StringValue(strings.Join(parts, ","))
		}
	}

	for key, value := range newIdentityOidcAuth.ClaimMetadataMapping {
		claimMetadataMappingElements[key] = types.StringValue(value)
	}

	boundClaimsMapValue, diags := types.MapValue(types.StringType, boundClaimsElements)
	diagnose.Append(diags...)
	if diagnose.HasError() {
		return
	}

	claimMetadataMappingMapValue, diags := types.MapValue(types.StringType, claimMetadataMappingElements)
	diagnose.Append(diags...)
	if diagnose.HasError() {
		return
	}

	plan.BoundClaims = boundClaimsMapValue
	plan.ClaimMetadataMapping = claimMetadataMappingMapValue

	plan.BoundAudiences, diags = types.ListValueFrom(ctx, types.StringType, infisicalstrings.StringSplitAndTrim(newIdentityOidcAuth.BoundAudiences, ","))
	diagnose.Append(diags...)
	if diagnose.HasError() {
		return
	}

	planAccessTokenTrustedIps := make([]IdentityOidcAuthResourceTrustedIps, len(newIdentityOidcAuth.AccessTokenTrustedIPS))
	for i, el := range newIdentityOidcAuth.AccessTokenTrustedIPS {
		if el.Prefix != nil {
			planAccessTokenTrustedIps[i] = IdentityOidcAuthResourceTrustedIps{IpAddress: types.StringValue(
				el.IpAddress + "/" + strconv.Itoa(*el.Prefix),
			)}
		} else {
			planAccessTokenTrustedIps[i] = IdentityOidcAuthResourceTrustedIps{IpAddress: types.StringValue(
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
}

// Create creates the resource and sets the initial Terraform state.
func (r *IdentityOidcAuthResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create identity oidc auth",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan IdentityOidcAuthResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	accessTokenTrustedIps := tfPlanExpandIpFieldAsApiField(ctx, resp.Diagnostics, plan.AccessTokenTrustedIps)
	boundAudiences := terraform.StringListToGoStringSlice(ctx, resp.Diagnostics, plan.BoundAudiences)

	boundClaimsMap := make(map[string]string)
	for key, value := range plan.BoundClaims.Elements() {
		if strVal, ok := value.(types.String); ok {
			boundClaimsMap[key] = strVal.ValueString()
		} else {
			resp.Diagnostics.AddError(
				"Error creating identity oidc auth",
				"Bound claims value is not a string",
			)
			return
		}
	}

	claimMetadataMappingMap := make(map[string]string)
	for key, value := range plan.ClaimMetadataMapping.Elements() {
		if strVal, ok := value.(types.String); ok {
			claimMetadataMappingMap[key] = strVal.ValueString()
		}
	}

	var newIdentityOidcAuth infisical.IdentityOidcAuth
	var err error
	if !plan.TemplateID.IsNull() {
		newIdentityOidcAuth, err = r.client.CreateIdentityOidcAuthFromTemplate(infisical.CreateIdentityOidcAuthFromTemplateRequest{
			IdentityID:              plan.IdentityID.ValueString(),
			TemplateID:              plan.TemplateID.ValueString(),
			AccessTokenTTL:          plan.AccessTokenTTL.ValueInt64(),
			AccessTokenMaxTTL:       plan.AccessTokenMaxTTL.ValueInt64(),
			AccessTokenNumUsesLimit: plan.AccessTokenNumUsesLimit.ValueInt64(),
			AccessTokenTrustedIPS:   accessTokenTrustedIps,
			BoundSubject:            plan.BoundSubject.ValueString(),
			BoundClaims:             boundClaimsMap,
			ClaimMetadataMapping:    claimMetadataMappingMap,
		})
	} else {
		newIdentityOidcAuth, err = r.client.CreateIdentityOidcAuth(infisical.CreateIdentityOidcAuthRequest{
			IdentityID:              plan.IdentityID.ValueString(),
			AccessTokenTTL:          plan.AccessTokenTTL.ValueInt64(),
			AccessTokenMaxTTL:       plan.AccessTokenMaxTTL.ValueInt64(),
			AccessTokenNumUsesLimit: plan.AccessTokenNumUsesLimit.ValueInt64(),
			AccessTokenTrustedIPS:   accessTokenTrustedIps,
			OidcDiscoveryUrl:        plan.OidcDiscoveryUrl.ValueString(),
			BoundAudiences:          strings.Join(boundAudiences, ","),
			BoundIssuer:             plan.BoundIssuer.ValueString(),
			BoundSubject:            plan.BoundSubject.ValueString(),
			BoundClaims:             boundClaimsMap,
			ClaimMetadataMapping:    claimMetadataMappingMap,
			CACERT:                  plan.CaCertificate.ValueString(),
		})
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating identity oidc auth",
			"Couldn't save oidc auth to Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(newIdentityOidcAuth.ID)
	updateOidcAuthStateByApi(ctx, resp.Diagnostics, &plan, &newIdentityOidcAuth)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *IdentityOidcAuthResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to get identity oidc auth",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state IdentityOidcAuthResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the latest data from the API
	identityOidcAuth, err := r.client.GetIdentityOidcAuth(infisical.GetIdentityOidcAuthRequest{
		IdentityID: state.IdentityID.ValueString(),
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError(
				"Error reading identity oidc auth",
				"Couldn't read identity oidc auth from Infisical, unexpected error: "+err.Error(),
			)
			return
		}
	}

	updateOidcAuthStateByApi(ctx, resp.Diagnostics, &state, &identityOidcAuth)
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *IdentityOidcAuthResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update identity oidc auth",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan IdentityOidcAuthResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state IdentityOidcAuthResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	accessTokenTrustedIps := tfPlanExpandIpFieldAsApiField(ctx, resp.Diagnostics, plan.AccessTokenTrustedIps)
	boundAudiences := terraform.StringListToGoStringSlice(ctx, resp.Diagnostics, plan.BoundAudiences)

	boundClaimsMap := make(map[string]string)
	for key, value := range plan.BoundClaims.Elements() {
		if strVal, ok := value.(types.String); ok {
			boundClaimsMap[key] = strVal.ValueString()
		} else {
			resp.Diagnostics.AddError(
				"Error updating identity oidc auth",
				"Bound claims value is not a string",
			)
			return
		}
	}

	claimMetadataMappingMap := make(map[string]string)
	for key, value := range plan.ClaimMetadataMapping.Elements() {
		if strVal, ok := value.(types.String); ok {
			claimMetadataMappingMap[key] = strVal.ValueString()
		}
	}

	var updatedIdentityOidcAuth infisical.IdentityOidcAuth
	var err error
	if oidcAuthUnlinksOnly(plan, state) {
		updatedIdentityOidcAuth, err = r.unlinkTemplateOnly(state)
	} else if !plan.TemplateID.IsNull() {
		updatedIdentityOidcAuth, err = r.client.UpdateIdentityOidcAuthFromTemplate(infisical.UpdateIdentityOidcAuthFromTemplateRequest{
			IdentityID:              plan.IdentityID.ValueString(),
			TemplateID:              plan.TemplateID.ValueString(),
			AccessTokenTTL:          plan.AccessTokenTTL.ValueInt64(),
			AccessTokenMaxTTL:       plan.AccessTokenMaxTTL.ValueInt64(),
			AccessTokenNumUsesLimit: plan.AccessTokenNumUsesLimit.ValueInt64(),
			AccessTokenTrustedIPS:   accessTokenTrustedIps,
			BoundSubject:            plan.BoundSubject.ValueString(),
			BoundClaims:             boundClaimsMap,
			ClaimMetadataMapping:    claimMetadataMappingMap,
		})
	} else {
		updatedIdentityOidcAuth, err = r.client.UpdateIdentityOidcAuth(infisical.UpdateIdentityOidcAuthRequest{
			IdentityID:              plan.IdentityID.ValueString(),
			AccessTokenTTL:          plan.AccessTokenTTL.ValueInt64(),
			AccessTokenMaxTTL:       plan.AccessTokenMaxTTL.ValueInt64(),
			AccessTokenNumUsesLimit: plan.AccessTokenNumUsesLimit.ValueInt64(),
			AccessTokenTrustedIPS:   accessTokenTrustedIps,
			OidcDiscoveryUrl:        plan.OidcDiscoveryUrl.ValueString(),
			BoundAudiences:          strings.Join(boundAudiences, ","),
			BoundIssuer:             plan.BoundIssuer.ValueString(),
			BoundSubject:            plan.BoundSubject.ValueString(),
			BoundClaims:             boundClaimsMap,
			ClaimMetadataMapping:    claimMetadataMappingMap,
			CACERT:                  plan.CaCertificate.ValueString(),
			// Null unlinks a template this identity was linked to.
			TemplateID: nil,
		})
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating identity oidc auth",
			"Couldn't update identity oidc auth from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	updateOidcAuthStateByApi(ctx, resp.Diagnostics, &plan, &updatedIdentityOidcAuth)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// oidcAuthUnlinksOnly reports whether the plan removes template_id and changes nothing else, which
// the configuration expresses by setting the template's values. Such a plan needs only the link
// removed, not an edit of the auth configuration.
func oidcAuthUnlinksOnly(plan, state IdentityOidcAuthResourceModel) bool {
	if state.TemplateID.IsNull() || !plan.TemplateID.IsNull() {
		return false
	}

	return plan.OidcDiscoveryUrl.Equal(state.OidcDiscoveryUrl) &&
		plan.BoundIssuer.Equal(state.BoundIssuer) &&
		plan.BoundAudiences.Equal(state.BoundAudiences) &&
		!plan.CaCertificate.IsUnknown() &&
		strings.TrimSpace(plan.CaCertificate.ValueString()) == strings.TrimSpace(state.CaCertificate.ValueString()) &&
		plan.BoundSubject.Equal(state.BoundSubject) &&
		plan.BoundClaims.Equal(state.BoundClaims) &&
		plan.ClaimMetadataMapping.Equal(state.ClaimMetadataMapping) &&
		// Unset TTLs and trusted IPs are left out of an update, so the API keeps them.
		unchangedOrDefaulted(plan.AccessTokenTTL, state.AccessTokenTTL) &&
		unchangedOrDefaulted(plan.AccessTokenMaxTTL, state.AccessTokenMaxTTL) &&
		unchangedOrDefaulted(plan.AccessTokenNumUsesLimit, state.AccessTokenNumUsesLimit) &&
		unchangedOrDefaulted(plan.AccessTokenTrustedIps, state.AccessTokenTrustedIps)
}

// unlinkTemplateOnly removes the template link and nothing else. It goes through the template's
// delete-usage endpoint first, which needs only unlink-templates, so a role scoped to unlinking can
// do it. That endpoint also requires the template to exist and the org's plan to include
// templates, so on any failure it falls back to the identity's own update endpoint, which needs
// edit-auth and has neither requirement. Either way the identity keeps the settings copied onto it.
func (r *IdentityOidcAuthResource) unlinkTemplateOnly(state IdentityOidcAuthResourceModel) (infisical.IdentityOidcAuth, error) {
	identityID := state.IdentityID.ValueString()

	usageErr := r.client.UnlinkIdentityAuthTemplateUsage(state.TemplateID.ValueString(), []string{identityID})
	if usageErr != nil {
		if _, err := r.client.UnlinkIdentityOidcAuthTemplate(identityID); err != nil {
			return infisical.IdentityOidcAuth{}, fmt.Errorf(
				"unlinking needs unlink-templates on auth templates or edit-auth on the identity; the template's delete-usage endpoint failed (%w) and so did the identity's update endpoint (%w)",
				usageErr, err,
			)
		}
	}

	return r.client.GetIdentityOidcAuth(infisical.GetIdentityOidcAuthRequest{IdentityID: identityID})
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *IdentityOidcAuthResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete identity oidc auth",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state IdentityOidcAuthResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.RevokeIdentityOidcAuth(infisical.RevokeIdentityOidcAuthRequest{
		IdentityID: state.IdentityID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting identity oidc auth",
			"Couldn't delete identity oidc auth from Infisical, unexpected error: "+err.Error(),
		)
		return
	}
}
