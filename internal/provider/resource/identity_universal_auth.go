package resource

import (
	"context"
	"fmt"
	"strconv"
	infisical "terraform-provider-infisical/internal/client"
	infisicalclient "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NewIdentityUniversalAuthResource is a helper function to simplify the provider implementation.
func NewIdentityUniversalAuthResource() resource.Resource {
	return &IdentityUniversalAuthResource{}
}

// IdentityUniversalAuthResource is the resource implementation.
type IdentityUniversalAuthResource struct {
	client *infisical.Client
}

// IdentityUniversalAuthResourceSourceModel describes the data source data model.
type IdentityUniversalAuthResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	IdentityID              types.String `tfsdk:"identity_id"`
	ClientSecretTrustedIps  types.List   `tfsdk:"client_secret_trusted_ips"`
	AccessTokenTrustedIps   types.List   `tfsdk:"access_token_trusted_ips"`
	AccessTokenTTL          types.Int64  `tfsdk:"access_token_ttl"`
	AccessTokenMaxTTL       types.Int64  `tfsdk:"access_token_max_ttl"`
	AccessTokenNumUsesLimit types.Int64  `tfsdk:"access_token_num_uses_limit"`
}

type IdentityUniversalAuthResourceTrustedIps struct {
	IpAddress types.String `tfsdk:"ip_address"`
}

// Metadata returns the resource type name.
func (r *IdentityUniversalAuthResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_universal_auth"
}

// Schema defines the schema for the resource.
func (r *IdentityUniversalAuthResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage identity universal auth in Infisical.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the universal auth",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"identity_id": schema.StringAttribute{
				Description:   "The ID of the identity to attach the configuration onto.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"client_secret_trusted_ips": schema.ListNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "A list of IPs or CIDR ranges that the Client Secret can be used from together with the Client ID to get back an access token. You can use 0.0.0.0/0, to allow usage from any network address.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ip_address": schema.StringAttribute{
							Optional: true,
							Computed: true,
						},
					},
				},
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
func (r *IdentityUniversalAuthResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func updateUniversalAuthStateByApi(ctx context.Context, diagnose diag.Diagnostics, plan *IdentityUniversalAuthResourceModel, newIdentityUniversalAuth *infisicalclient.IdentityUniversalAuth) {
	plan.AccessTokenMaxTTL = types.Int64Value(newIdentityUniversalAuth.AccessTokenMaxTTL)
	plan.AccessTokenTTL = types.Int64Value(newIdentityUniversalAuth.AccessTokenTTL)
	plan.AccessTokenNumUsesLimit = types.Int64Value(newIdentityUniversalAuth.AccessTokenNumUsesLimit)

	planClientSecretTrustedIps := make([]IdentityUniversalAuthResourceTrustedIps, len(newIdentityUniversalAuth.ClientSecretTrustedIps))
	for i, el := range newIdentityUniversalAuth.ClientSecretTrustedIps {
		if el.Prefix != nil {
			planClientSecretTrustedIps[i] = IdentityUniversalAuthResourceTrustedIps{IpAddress: types.StringValue(
				el.IpAddress + "/" + strconv.Itoa(*el.Prefix),
			)}
		} else {
			planClientSecretTrustedIps[i] = IdentityUniversalAuthResourceTrustedIps{IpAddress: types.StringValue(
				el.IpAddress,
			)}
		}
	}

	planAccessTokenTrustedIps := make([]IdentityUniversalAuthResourceTrustedIps, len(newIdentityUniversalAuth.AccessTokenTrustedIps))
	for i, el := range newIdentityUniversalAuth.AccessTokenTrustedIps {
		if el.Prefix != nil {
			planAccessTokenTrustedIps[i] = IdentityUniversalAuthResourceTrustedIps{IpAddress: types.StringValue(
				el.IpAddress + "/" + strconv.Itoa(*el.Prefix),
			)}
		} else {
			planAccessTokenTrustedIps[i] = IdentityUniversalAuthResourceTrustedIps{IpAddress: types.StringValue(
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

	stateClientSecretTrustedIps, diags := types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"ip_address": types.StringType,
		},
	}, planClientSecretTrustedIps)

	diagnose.Append(diags...)
	if diagnose.HasError() {
		return
	}

	plan.AccessTokenTrustedIps = stateAccessTokenTrustedIps
	plan.ClientSecretTrustedIps = stateClientSecretTrustedIps
}

func tfPlanExpandIpFieldAsApiField(ctx context.Context, diagnostics diag.Diagnostics, planField types.List) []infisical.IdentityAuthTrustedIpRequest {
	var planAccessTokenTrustedIps []IdentityAwsAuthResourceTrustedIps
	diags := planField.ElementsAs(ctx, &planAccessTokenTrustedIps, false)
	diagnostics.Append(diags...)
	if diagnostics.HasError() {
		return nil
	}
	trustedIps := make([]infisical.IdentityAuthTrustedIpRequest, len(planAccessTokenTrustedIps))
	for i, ip := range planAccessTokenTrustedIps {
		trustedIps[i] = infisical.IdentityAuthTrustedIpRequest{
			IPAddress: ip.IpAddress.ValueString(),
		}
	}
	return trustedIps
}

// Create creates the resource and sets the initial Terraform state.
func (r *IdentityUniversalAuthResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client.Config.AuthStrategy != infisical.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY {
		resp.Diagnostics.AddError(
			"Unable to create identity universal auth",
			"Only Machine IdentityUniversalAuth authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan IdentityUniversalAuthResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	accessTokenTrustedIps := tfPlanExpandIpFieldAsApiField(ctx, resp.Diagnostics, plan.AccessTokenTrustedIps)
	clientSecretTrustedIps := tfPlanExpandIpFieldAsApiField(ctx, resp.Diagnostics, plan.ClientSecretTrustedIps)
	newIdentityUniversalAuth, err := r.client.CreateIdentityUniversalAuth(infisical.CreateIdentityUniversalAuthRequest{
		IdentityID:              plan.IdentityID.ValueString(),
		ClientSecretTrustedIPs:  clientSecretTrustedIps,
		AccessTokenTrustedIPs:   accessTokenTrustedIps,
		AccessTokenTTL:          plan.AccessTokenTTL.ValueInt64(),
		AccessTokenMaxTTL:       plan.AccessTokenMaxTTL.ValueInt64(),
		AccessTokenNumUsesLimit: plan.AccessTokenNumUsesLimit.ValueInt64(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating identity universal auth",
			"Couldn't save tag to Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(newIdentityUniversalAuth.ID)
	updateUniversalAuthStateByApi(ctx, resp.Diagnostics, &plan, &newIdentityUniversalAuth)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *IdentityUniversalAuthResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client.Config.AuthStrategy != infisical.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY {
		resp.Diagnostics.AddError(
			"Unable to read identity universal auth role",
			"Only Machine IdentityUniversalAuth authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state IdentityUniversalAuthResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the latest data from the API
	identityUniversalAuth, err := r.client.GetIdentityUniversalAuth(infisical.GetIdentityUniversalAuthRequest{
		IdentityID: state.IdentityID.ValueString(),
	})

	if err != nil {
		if err == infisicalclient.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError(
				"Error reading identity universal auth",
				"Couldn't read identity universal auth from Infiscial, unexpected error: "+err.Error(),
			)
			return
		}
	}

	updateUniversalAuthStateByApi(ctx, resp.Diagnostics, &state, &identityUniversalAuth)
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *IdentityUniversalAuthResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client.Config.AuthStrategy != infisical.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY {
		resp.Diagnostics.AddError(
			"Unable to update identity universal auth",
			"Only Machine IdentityUniversalAuth authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan IdentityUniversalAuthResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state IdentityUniversalAuthResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	accessTokenTrustedIps := tfPlanExpandIpFieldAsApiField(ctx, resp.Diagnostics, plan.AccessTokenTrustedIps)
	clientSecretTrustedIps := tfPlanExpandIpFieldAsApiField(ctx, resp.Diagnostics, plan.ClientSecretTrustedIps)
	updatedIdentityUniversalAuth, err := r.client.UpdateIdentityUniversalAuth(infisical.UpdateIdentityUniversalAuthRequest{
		IdentityID:              plan.IdentityID.ValueString(),
		ClientSecretTrustedIPs:  clientSecretTrustedIps,
		AccessTokenTrustedIPs:   accessTokenTrustedIps,
		AccessTokenTTL:          plan.AccessTokenTTL.ValueInt64(),
		AccessTokenMaxTTL:       plan.AccessTokenMaxTTL.ValueInt64(),
		AccessTokenNumUsesLimit: plan.AccessTokenNumUsesLimit.ValueInt64(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating identity universal auth",
			"Couldn't update identity universal auth from Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	updateUniversalAuthStateByApi(ctx, resp.Diagnostics, &plan, &updatedIdentityUniversalAuth)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *IdentityUniversalAuthResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	if r.client.Config.AuthStrategy != infisical.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY {
		resp.Diagnostics.AddError(
			"Unable to delete identity universal auth",
			"Only Machine IdentityUniversalAuth authentication is supported for this operation",
		)
		return
	}

	var state IdentityUniversalAuthResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.RevokeIdentityUniversalAuth(infisical.RevokeIdentityUniversalAuthRequest{
		IdentityID: state.IdentityID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting identity universal auth",
			"Couldn't delete identity universal auth from Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

}
