package resource

import (
	"context"
	"encoding/pem"
	"fmt"
	"strconv"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
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

func NewIdentityJwtAuthResource() resource.Resource {
	return &IdentityJwtAuthResource{}
}

type IdentityJwtAuthResource struct {
	client *infisical.Client
}

type IdentityJwtAuthResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	IdentityID              types.String `tfsdk:"identity_id"`
	ConfigurationType       types.String `tfsdk:"configuration_type"`
	JwksUrl                 types.String `tfsdk:"jwks_url"`
	JwksCaCert              types.String `tfsdk:"jwks_ca_cert"`
	PublicKeys              types.List   `tfsdk:"public_keys"`
	BoundIssuer             types.String `tfsdk:"bound_issuer"`
	BoundAudiences          types.List   `tfsdk:"bound_audiences"`
	BoundClaims             types.Map    `tfsdk:"bound_claims"`
	BoundSubject            types.String `tfsdk:"bound_subject"`
	AccessTokenTrustedIps   types.List   `tfsdk:"access_token_trusted_ips"`
	AccessTokenTTL          types.Int64  `tfsdk:"access_token_ttl"`
	AccessTokenMaxTTL       types.Int64  `tfsdk:"access_token_max_ttl"`
	AccessTokenNumUsesLimit types.Int64  `tfsdk:"access_token_num_uses_limit"`
}

type IdentityJwtAuthResourceTrustedIps struct {
	IpAddress types.String `tfsdk:"ip_address"`
}

func (r *IdentityJwtAuthResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_jwt_auth"
}

func (r *IdentityJwtAuthResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage identity JWT auth in Infisical.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the JWT auth.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"identity_id": schema.StringAttribute{
				Description:   "The ID of the identity to attach the configuration onto.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"configuration_type": schema.StringAttribute{
				Description: "The configuration type of the JWT auth. Must be 'jwks' or 'static'.",
				Required:    true,
				Validators:  []validator.String{stringvalidator.OneOf("jwks", "static")},
			},
			"jwks_url": schema.StringAttribute{
				Description:   "The URL used to retrieve the JSON Web Key Set (JWKS) for verifying JWTs. Required when configuration_type is 'jwks'.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"jwks_ca_cert": schema.StringAttribute{
				Description:   "The PEM-encoded CA certificate for validating the TLS connection to the JWKS URL.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"public_keys": schema.ListAttribute{
				Description:   "A list of PEM-encoded public keys used to verify JWTs. Required when configuration_type is 'static'.",
				Optional:      true,
				Computed:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"bound_issuer": schema.StringAttribute{
				Description:   "The unique identifier of the identity provider issuing the JWTs.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"bound_audiences": schema.ListAttribute{
				Description:   "The list of intended recipients.",
				Optional:      true,
				Computed:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"bound_claims": schema.MapAttribute{
				Description: "The attributes that should be present in the JWT for it to be valid.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
					pkg.CommaSpaceMapModifier{},
				},
			},
			"bound_subject": schema.StringAttribute{
				Description:   "The expected principal that is the subject of the JWT.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"access_token_trusted_ips": schema.ListNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "A list of IPs or CIDR ranges that access tokens can be used from. You can use 0.0.0.0/0, to allow usage from any network address.",
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
				Description:   "The maximum number of times that an access token can be used; a value of 0 implies infinite number of uses. Default: 0",
				Computed:      true,
				Optional:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *IdentityJwtAuthResource) validateCACert(caCert string, resp *resource.ValidateConfigResponse) {
	block, _ := pem.Decode([]byte(caCert))
	if block == nil {
		resp.Diagnostics.AddError(
			"Invalid JWT auth configuration",
			"jwks_ca_cert is not a valid PEM-encoded certificate",
		)
		return
	}
	if block.Type != "CERTIFICATE" {
		resp.Diagnostics.AddError(
			"Invalid JWT auth configuration",
			fmt.Sprintf("jwks_ca_cert has unexpected PEM block type %q, expected \"CERTIFICATE\"", block.Type),
		)
	}
}

func (r *IdentityJwtAuthResource) validatePEM(pemKeys []string, resp *resource.ValidateConfigResponse) {
	for i, key := range pemKeys {
		block, _ := pem.Decode([]byte(key))
		if block == nil {
			resp.Diagnostics.AddError(
				"Invalid JWT auth configuration",
				fmt.Sprintf("public_keys[%d] is not a valid PEM-encoded key", i),
			)
			return
		}
	}
}

func (r *IdentityJwtAuthResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config IdentityJwtAuthResourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	configurationType := config.ConfigurationType.ValueString()

	if configurationType == "jwks" {
		if config.JwksUrl.IsNull() || config.JwksUrl.IsUnknown() || config.JwksUrl.ValueString() == "" {
			resp.Diagnostics.AddError(
				"Invalid JWT auth configuration",
				"jwks_url is required when configuration_type is 'jwks'",
			)
		}

		if !config.JwksCaCert.IsNull() && !config.JwksCaCert.IsUnknown() && config.JwksCaCert.ValueString() != "" {
			r.validateCACert(config.JwksCaCert.ValueString(), resp)
		}

		if resp.Diagnostics.HasError() {
			return
		}
	}

	if configurationType == "static" {
		if config.PublicKeys.IsNull() || config.PublicKeys.IsUnknown() || len(config.PublicKeys.Elements()) == 0 {
			resp.Diagnostics.AddError(
				"Invalid JWT auth configuration",
				"public_keys must have at least one entry when configuration_type is 'static'",
			)
			return
		}
		publicKeys := terraform.StringListToGoStringSlice(ctx, resp.Diagnostics, config.PublicKeys)
		r.validatePEM(publicKeys, resp)
		if resp.Diagnostics.HasError() {
			return
		}
	}
}

func (r *IdentityJwtAuthResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func updateJwtAuthStateByApi(ctx context.Context, diagnose diag.Diagnostics, plan *IdentityJwtAuthResourceModel, newIdentityJwtAuth *infisical.IdentityJwtAuth) {
	plan.AccessTokenMaxTTL = types.Int64Value(newIdentityJwtAuth.AccessTokenMaxTTL)
	plan.AccessTokenTTL = types.Int64Value(newIdentityJwtAuth.AccessTokenTTL)
	plan.AccessTokenNumUsesLimit = types.Int64Value(newIdentityJwtAuth.AccessTokenNumUsesLimit)

	plan.ConfigurationType = types.StringValue(newIdentityJwtAuth.ConfigurationType)
	plan.JwksUrl = types.StringValue(newIdentityJwtAuth.JwksUrl)
	plan.JwksCaCert = types.StringValue(newIdentityJwtAuth.JwksCaCert)
	plan.BoundIssuer = types.StringValue(newIdentityJwtAuth.BoundIssuer)
	plan.BoundSubject = types.StringValue(newIdentityJwtAuth.BoundSubject)

	// Public keys
	var diags diag.Diagnostics
	publicKeys := newIdentityJwtAuth.PublicKeys
	if publicKeys == nil {
		publicKeys = []string{}
	}
	plan.PublicKeys, diags = types.ListValueFrom(ctx, types.StringType, publicKeys)
	diagnose.Append(diags...)
	if diagnose.HasError() {
		return
	}

	// Bound claims
	if len(newIdentityJwtAuth.BoundClaims) == 0 {
		if plan.BoundClaims.IsNull() {
			plan.BoundClaims = types.MapNull(types.StringType)
		} else {
			plan.BoundClaims, diags = types.MapValue(types.StringType, map[string]attr.Value{})
			diagnose.Append(diags...)
			if diagnose.HasError() {
				return
			}
		}
	} else {
		boundClaimsElements := make(map[string]attr.Value)
		for key, value := range newIdentityJwtAuth.BoundClaims {
			useSpaces := false
			if !plan.BoundClaims.IsNull() {
				if planValue, ok := plan.BoundClaims.Elements()[key]; ok {
					if planStr, ok := planValue.(types.String); ok {
						useSpaces = strings.Contains(planStr.ValueString(), ", ")
					}
				}
			}

			parts := strings.Split(value, ",")
			for i, part := range parts {
				parts[i] = strings.TrimSpace(part)
			}

			if useSpaces {
				boundClaimsElements[key] = types.StringValue(strings.Join(parts, ", "))
			} else {
				boundClaimsElements[key] = types.StringValue(strings.Join(parts, ","))
			}
		}

		boundClaimsMapValue, diags2 := types.MapValue(types.StringType, boundClaimsElements)
		diagnose.Append(diags2...)
		if diagnose.HasError() {
			return
		}
		plan.BoundClaims = boundClaimsMapValue
	}

	// Bound audiences
	var boundAudiencesList []string
	if newIdentityJwtAuth.BoundAudiences != "" {
		boundAudiencesList = infisicalstrings.StringSplitAndTrim(newIdentityJwtAuth.BoundAudiences, ",")
	} else {
		boundAudiencesList = []string{}
	}
	plan.BoundAudiences, diags = types.ListValueFrom(ctx, types.StringType, boundAudiencesList)
	diagnose.Append(diags...)
	if diagnose.HasError() {
		return
	}

	// Access token trusted IPs
	planAccessTokenTrustedIps := make([]IdentityJwtAuthResourceTrustedIps, len(newIdentityJwtAuth.AccessTokenTrustedIPS))
	for i, el := range newIdentityJwtAuth.AccessTokenTrustedIPS {
		if el.Prefix != nil {
			planAccessTokenTrustedIps[i] = IdentityJwtAuthResourceTrustedIps{IpAddress: types.StringValue(
				el.IpAddress + "/" + strconv.Itoa(*el.Prefix),
			)}
		} else {
			planAccessTokenTrustedIps[i] = IdentityJwtAuthResourceTrustedIps{IpAddress: types.StringValue(
				el.IpAddress,
			)}
		}
	}

	stateAccessTokenTrustedIps, diags3 := types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"ip_address": types.StringType,
		},
	}, planAccessTokenTrustedIps)

	diagnose.Append(diags3...)
	if diagnose.HasError() {
		return
	}

	plan.AccessTokenTrustedIps = stateAccessTokenTrustedIps
}

func (r *IdentityJwtAuthResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create identity JWT auth",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan IdentityJwtAuthResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	configurationType := plan.ConfigurationType.ValueString()

	accessTokenTrustedIps := tfPlanExpandIpFieldAsApiField(ctx, resp.Diagnostics, plan.AccessTokenTrustedIps)
	boundAudiences := terraform.StringListToGoStringSlice(ctx, resp.Diagnostics, plan.BoundAudiences)
	publicKeys := terraform.StringListToGoStringSlice(ctx, resp.Diagnostics, plan.PublicKeys)

	boundClaimsMap := make(map[string]string)
	for key, value := range plan.BoundClaims.Elements() {
		if strVal, ok := value.(types.String); ok {
			boundClaimsMap[key] = strVal.ValueString()
		} else {
			resp.Diagnostics.AddError(
				"Error creating identity JWT auth",
				"Bound claims value is not a string",
			)
			return
		}
	}

	newIdentityJwtAuth, err := r.client.CreateIdentityJwtAuth(infisical.IdentityJwtAuthRequest{
		IdentityID:              plan.IdentityID.ValueString(),
		ConfigurationType:       configurationType,
		JwksUrl:                 plan.JwksUrl.ValueString(),
		JwksCaCert:              plan.JwksCaCert.ValueString(),
		PublicKeys:              publicKeys,
		BoundIssuer:             plan.BoundIssuer.ValueString(),
		BoundAudiences:          strings.Join(boundAudiences, ","),
		BoundClaims:             boundClaimsMap,
		BoundSubject:            plan.BoundSubject.ValueString(),
		AccessTokenTrustedIPS:   accessTokenTrustedIps,
		AccessTokenTTL:          plan.AccessTokenTTL.ValueInt64(),
		AccessTokenMaxTTL:       plan.AccessTokenMaxTTL.ValueInt64(),
		AccessTokenNumUsesLimit: plan.AccessTokenNumUsesLimit.ValueInt64(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating identity JWT auth",
			"Couldn't save JWT auth to Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(newIdentityJwtAuth.ID)
	updateJwtAuthStateByApi(ctx, resp.Diagnostics, &plan, &newIdentityJwtAuth)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *IdentityJwtAuthResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to get identity JWT auth",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state IdentityJwtAuthResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	identityJwtAuth, err := r.client.GetIdentityJwtAuth(infisical.GetIdentityJwtAuthRequest{
		IdentityID: state.IdentityID.ValueString(),
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError(
				"Error reading identity JWT auth",
				"Couldn't read identity JWT auth from Infisical, unexpected error: "+err.Error(),
			)
			return
		}
	}

	updateJwtAuthStateByApi(ctx, resp.Diagnostics, &state, &identityJwtAuth)
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *IdentityJwtAuthResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update identity JWT auth",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var plan IdentityJwtAuthResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state IdentityJwtAuthResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	configurationType := plan.ConfigurationType.ValueString()

	accessTokenTrustedIps := tfPlanExpandIpFieldAsApiField(ctx, resp.Diagnostics, plan.AccessTokenTrustedIps)
	boundAudiences := terraform.StringListToGoStringSlice(ctx, resp.Diagnostics, plan.BoundAudiences)
	publicKeys := terraform.StringListToGoStringSlice(ctx, resp.Diagnostics, plan.PublicKeys)

	boundClaimsMap := make(map[string]string)
	for key, value := range plan.BoundClaims.Elements() {
		if strVal, ok := value.(types.String); ok {
			boundClaimsMap[key] = strVal.ValueString()
		} else {
			resp.Diagnostics.AddError(
				"Error updating identity JWT auth",
				"Bound claims value is not a string",
			)
			return
		}
	}

	updatedIdentityJwtAuth, err := r.client.UpdateIdentityJwtAuth(infisical.IdentityJwtAuthRequest{
		IdentityID:              plan.IdentityID.ValueString(),
		ConfigurationType:       configurationType,
		JwksUrl:                 plan.JwksUrl.ValueString(),
		JwksCaCert:              plan.JwksCaCert.ValueString(),
		PublicKeys:              publicKeys,
		BoundIssuer:             plan.BoundIssuer.ValueString(),
		BoundAudiences:          strings.Join(boundAudiences, ","),
		BoundClaims:             boundClaimsMap,
		BoundSubject:            plan.BoundSubject.ValueString(),
		AccessTokenTrustedIPS:   accessTokenTrustedIps,
		AccessTokenTTL:          plan.AccessTokenTTL.ValueInt64(),
		AccessTokenMaxTTL:       plan.AccessTokenMaxTTL.ValueInt64(),
		AccessTokenNumUsesLimit: plan.AccessTokenNumUsesLimit.ValueInt64(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating identity JWT auth",
			"Couldn't update identity JWT auth from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	updateJwtAuthStateByApi(ctx, resp.Diagnostics, &plan, &updatedIdentityJwtAuth)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *IdentityJwtAuthResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete identity JWT auth",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state IdentityJwtAuthResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.RevokeIdentityJwtAuth(infisical.RevokeIdentityJwtAuthRequest{
		IdentityID: state.IdentityID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting identity JWT auth",
			"Couldn't delete identity JWT auth from Infisical, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *IdentityJwtAuthResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to import identity JWT auth",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	resource.ImportStatePassthroughID(ctx, path.Root("identity_id"), req, resp)
}
