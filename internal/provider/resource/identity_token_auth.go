package resource

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NewIdentityTokenAuthResource is a helper function to simplify the provider implementation.
func NewIdentityTokenAuthResource() resource.Resource {
	return &IdentityTokenAuthResource{}
}

// IdentityTokenAuthResource is the resource implementation.
type IdentityTokenAuthResource struct {
	client *infisical.Client
}

// IdentityTokenAuthResourceModel describes the data source data model.
type IdentityTokenAuthResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	IdentityID              types.String `tfsdk:"identity_id"`
	AccessTokenTrustedIps   types.List   `tfsdk:"access_token_trusted_ips"`
	AccessTokenTTL          types.Int64  `tfsdk:"access_token_ttl"`
	AccessTokenMaxTTL       types.Int64  `tfsdk:"access_token_max_ttl"`
	AccessTokenNumUsesLimit types.Int64  `tfsdk:"access_token_num_uses_limit"`
}

type IdentityTokenAuthResourceTrustedIps struct {
	IpAddress types.String `tfsdk:"ip_address"`
}

// Metadata returns the resource type name.
func (r *IdentityTokenAuthResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_token_auth"
}

// Schema defines the schema for the resource.
func (r *IdentityTokenAuthResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage identity token auth in Infisical.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the token auth.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"identity_id": schema.StringAttribute{
				Description:   "The ID of the identity to attach the configuration onto.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
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

// Configure adds the provider configured client to the resource.
func (r *IdentityTokenAuthResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func updateTokenAuthStateByApi(ctx context.Context, diagnose diag.Diagnostics, plan *IdentityTokenAuthResourceModel, newIdentityTokenAuth *infisical.IdentityTokenAuth) {
	plan.AccessTokenMaxTTL = types.Int64Value(newIdentityTokenAuth.AccessTokenMaxTTL)
	plan.AccessTokenTTL = types.Int64Value(newIdentityTokenAuth.AccessTokenTTL)
	plan.AccessTokenNumUsesLimit = types.Int64Value(newIdentityTokenAuth.AccessTokenNumUsesLimit)

	planAccessTokenTrustedIps := make([]IdentityTokenAuthResourceTrustedIps, len(newIdentityTokenAuth.AccessTokenTrustedIPS))
	for i, el := range newIdentityTokenAuth.AccessTokenTrustedIPS {
		if el.Prefix != nil {
			planAccessTokenTrustedIps[i] = IdentityTokenAuthResourceTrustedIps{IpAddress: types.StringValue(
				el.IpAddress + "/" + strconv.Itoa(*el.Prefix),
			)}
		} else {
			planAccessTokenTrustedIps[i] = IdentityTokenAuthResourceTrustedIps{IpAddress: types.StringValue(
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
func (r *IdentityTokenAuthResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create identity token auth",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan IdentityTokenAuthResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	accessTokenTrustedIps := tfPlanExpandIpFieldAsApiField(ctx, resp.Diagnostics, plan.AccessTokenTrustedIps)

	newIdentityTokenAuth, err := r.client.CreateIdentityTokenAuth(infisical.CreateIdentityTokenAuthRequest{
		IdentityID:              plan.IdentityID.ValueString(),
		AccessTokenTTL:          plan.AccessTokenTTL.ValueInt64(),
		AccessTokenMaxTTL:       plan.AccessTokenMaxTTL.ValueInt64(),
		AccessTokenNumUsesLimit: plan.AccessTokenNumUsesLimit.ValueInt64(),
		AccessTokenTrustedIPS:   accessTokenTrustedIps,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating identity token auth",
			"Couldn't save token auth to Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(newIdentityTokenAuth.ID)
	updateTokenAuthStateByApi(ctx, resp.Diagnostics, &plan, &newIdentityTokenAuth)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *IdentityTokenAuthResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to get identity token auth",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state IdentityTokenAuthResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the latest data from the API
	identityTokenAuth, err := r.client.GetIdentityTokenAuth(infisical.GetIdentityTokenAuthRequest{
		IdentityID: state.IdentityID.ValueString(),
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError(
				"Error reading identity token auth",
				"Couldn't read identity token auth from Infiscial, unexpected error: "+err.Error(),
			)
			return
		}
	}

	updateTokenAuthStateByApi(ctx, resp.Diagnostics, &state, &identityTokenAuth)
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *IdentityTokenAuthResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update identity token auth",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan IdentityTokenAuthResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state IdentityTokenAuthResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	accessTokenTrustedIps := tfPlanExpandIpFieldAsApiField(ctx, resp.Diagnostics, plan.AccessTokenTrustedIps)

	updatedIdentityTokenAuth, err := r.client.UpdateIdentityTokenAuth(infisical.UpdateIdentityTokenAuthRequest{
		IdentityID:              plan.IdentityID.ValueString(),
		AccessTokenTTL:          plan.AccessTokenTTL.ValueInt64(),
		AccessTokenMaxTTL:       plan.AccessTokenMaxTTL.ValueInt64(),
		AccessTokenNumUsesLimit: plan.AccessTokenNumUsesLimit.ValueInt64(),
		AccessTokenTrustedIPS:   accessTokenTrustedIps,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating identity token auth",
			"Couldn't update identity token auth from Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	updateTokenAuthStateByApi(ctx, resp.Diagnostics, &plan, &updatedIdentityTokenAuth)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *IdentityTokenAuthResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete identity token auth",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state IdentityTokenAuthResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.RevokeIdentityTokenAuth(infisical.RevokeIdentityTokenAuthRequest{
		IdentityID: state.IdentityID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting identity token auth",
			"Couldn't delete identity token auth from Infiscial, unexpected error: "+err.Error(),
		)
		return
	}
}

// ImportState imports the resource using the identity ID.
func (r *IdentityTokenAuthResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to import identity token auth",
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
				"Error importing identity token auth",
				"Couldn't read identity from Infisical, unexpected error: "+err.Error(),
			)
		}
		return
	}

	if len(identity.Identity.AuthMethods) == 0 {
		resp.Diagnostics.AddError(
			"Identity token auth not found",
			"The identity with the given ID has no configured auth methods",
		)
		return
	}

	hasTokenAuth := slices.Contains(identity.Identity.AuthMethods, "token-auth")

	if !hasTokenAuth {
		resp.Diagnostics.AddError(
			"Identity token auth not found",
			"The identity with the given ID does not have token auth configured",
		)
		return
	}

	identityTokenAuth, err := r.client.GetIdentityTokenAuth(infisical.GetIdentityTokenAuthRequest{
		IdentityID: req.ID,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing identity token auth",
			"Couldn't read identity token auth from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	var diags diag.Diagnostics
	var state IdentityTokenAuthResourceModel

	state.ID = types.StringValue(identityTokenAuth.ID)
	state.IdentityID = types.StringValue(identityTokenAuth.IdentityID)
	state.AccessTokenTTL = types.Int64Value(identityTokenAuth.AccessTokenTTL)
	state.AccessTokenMaxTTL = types.Int64Value(identityTokenAuth.AccessTokenMaxTTL)
	state.AccessTokenNumUsesLimit = types.Int64Value(identityTokenAuth.AccessTokenNumUsesLimit)

	accessTokenTrustedIps := make([]IdentityTokenAuthResourceTrustedIps, len(identityTokenAuth.AccessTokenTrustedIPS))
	for i, el := range identityTokenAuth.AccessTokenTrustedIPS {
		if el.Prefix != nil {
			accessTokenTrustedIps[i] = IdentityTokenAuthResourceTrustedIps{IpAddress: types.StringValue(
				el.IpAddress + "/" + strconv.Itoa(*el.Prefix),
			)}
		} else {
			accessTokenTrustedIps[i] = IdentityTokenAuthResourceTrustedIps{IpAddress: types.StringValue(
				el.IpAddress,
			)}
		}
	}

	state.AccessTokenTrustedIps, diags = types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"ip_address": types.StringType,
		},
	}, accessTokenTrustedIps)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
