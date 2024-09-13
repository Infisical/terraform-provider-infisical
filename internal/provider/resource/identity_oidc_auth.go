package resource

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
	infisicalclient "terraform-provider-infisical/internal/client"
	infisicalstrings "terraform-provider-infisical/internal/pkg/strings"
	"terraform-provider-infisical/internal/pkg/terraform"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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
	ID                      types.String `tfsdk:"id"`
	IdentityID              types.String `tfsdk:"identity_id"`
	OidcDiscoveryUrl        types.String `tfsdk:"oidc_discovery_url"`
	CaCertificate           types.String `tfsdk:"oidc_ca_certificate"`
	BoundIssuer             types.String `tfsdk:"bound_issuer"`
	BoundAudiences          types.List   `tfsdk:"bound_audiences"`
	BoundClaims             types.Map    `tfsdk:"bound_claims"`
	BoundSubject            types.String `tfsdk:"bound_subject"`
	AccessTokenTrustedIps   types.List   `tfsdk:"access_token_trusted_ips"`
	AccessTokenTTL          types.Int64  `tfsdk:"access_token_ttl"`
	AccessTokenMaxTTL       types.Int64  `tfsdk:"access_token_max_ttl"`
	AccessTokenNumUsesLimit types.Int64  `tfsdk:"access_token_num_uses_limit"`
}

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
			"oidc_discovery_url": schema.StringAttribute{
				Description: "The URL used to retrieve the OpenID Connect configuration from the identity provider.",
				Required:    true,
			},
			"bound_issuer": schema.StringAttribute{
				Description: "The unique identifier of the identity provider issuing the OIDC tokens.",
				Required:    true,
			},
			"bound_audiences": schema.ListAttribute{
				Description: "The comma-separated list of intended recipients.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"bound_claims": schema.MapAttribute{
				Description: "The attributes that should be present in the JWT for it to be valid. The provided values can be a glob pattern.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"bound_subject": schema.StringAttribute{
				Description: "The expected principal that is the subject of the JWT.",
				Optional:    true,
				Computed:    true,
			},
			"oidc_ca_certificate": schema.StringAttribute{
				Description:         "The PEM-encoded CA cert for establishing secure communication with the Identity Provider endpoints",
				MarkdownDescription: "The PEM-encoded CA cert for establishing secure communication with the Identity Provider endpoints",
				Optional:            true,
				Computed:            true,
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

func updateOidcAuthStateByApi(ctx context.Context, diagnose diag.Diagnostics, plan *IdentityOidcAuthResourceModel, newIdentityOidcAuth *infisicalclient.IdentityOidcAuth) {
	plan.AccessTokenMaxTTL = types.Int64Value(newIdentityOidcAuth.AccessTokenMaxTTL)
	plan.AccessTokenTTL = types.Int64Value(newIdentityOidcAuth.AccessTokenTTL)
	plan.AccessTokenNumUsesLimit = types.Int64Value(newIdentityOidcAuth.AccessTokenNumUsesLimit)

	plan.OidcDiscoveryUrl = types.StringValue(newIdentityOidcAuth.OidcDiscoveryUrl)
	plan.BoundIssuer = types.StringValue(newIdentityOidcAuth.BoundIssuer)
	plan.BoundSubject = types.StringValue(newIdentityOidcAuth.BoundSubject)
	plan.CaCertificate = types.StringValue(newIdentityOidcAuth.CACERT)

	boundClaimsElements := make(map[string]attr.Value)
	for key, value := range newIdentityOidcAuth.BoundClaims {
		boundClaimsElements[key] = types.StringValue(value)
	}

	boundClaimsMapValue, diags := types.MapValue(types.StringType, boundClaimsElements)
	diagnose.Append(diags...)
	if diagnose.HasError() {
		return
	}

	plan.BoundClaims = boundClaimsMapValue

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

	newIdentityOidcAuth, err := r.client.CreateIdentityOidcAuth(infisical.CreateIdentityOidcAuthRequest{
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
		CACERT:                  plan.CaCertificate.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating identity oidc auth",
			"Couldn't save oidc auth to Infiscial, unexpected error: "+err.Error(),
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
		if err == infisicalclient.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError(
				"Error reading identity oidc auth",
				"Couldn't read identity oidc auth from Infiscial, unexpected error: "+err.Error(),
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

	updatedIdentityOidcAuth, err := r.client.UpdateIdentityOidcAuth(infisical.UpdateIdentityOidcAuthRequest{
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
		CACERT:                  plan.CaCertificate.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating identity oidc auth",
			"Couldn't update identity oidc auth from Infiscial, unexpected error: "+err.Error(),
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
			"Couldn't delete identity oidc auth from Infiscial, unexpected error: "+err.Error(),
		)
		return
	}
}
