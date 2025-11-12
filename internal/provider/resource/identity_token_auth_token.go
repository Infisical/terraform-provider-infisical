package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NewIdentityTokenAuthTokenResource is a helper function to simplify the provider implementation.
func NewIdentityTokenAuthTokenResource() resource.Resource {
	return &IdentityTokenAuthTokenResource{}
}

// IdentityTokenAuthTokenResource is the resource implementation.
type IdentityTokenAuthTokenResource struct {
	client *infisical.Client
}

// IdentityTokenAuthTokenResourceModel describes the resource data model.
type IdentityTokenAuthTokenResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	IdentityID        types.String `tfsdk:"identity_id"`
	NumberOfUses      types.Int64  `tfsdk:"number_of_uses"`
	NumberOfUsesLimit types.Int64  `tfsdk:"number_of_uses_limit"`
	TTL               types.Int64  `tfsdk:"ttl"`
	Token             types.String `tfsdk:"token"`
	CreatedAt         types.String `tfsdk:"created_at"`
	IsRevoked         types.Bool   `tfsdk:"is_revoked"`
}

// Metadata returns the resource type name.
func (r *IdentityTokenAuthTokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_token_auth_token"
}

// Schema defines the schema for the resource.
func (r *IdentityTokenAuthTokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage identity token auth token in Infisical.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the token auth token",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description:   "The name of the token auth token",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"identity_id": schema.StringAttribute{
				Description:   "The ID of the identity to create a token for",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"number_of_uses_limit": schema.Int64Attribute{
				Description: "The maximum number of times that the token can be used; a value of 0 implies infinite number of uses. Default: 0",
				Computed:    true,
			},
			"ttl": schema.Int64Attribute{
				Description: "The lifetime for the token in seconds. Default: 0 - not expiring",
				Computed:    true,
			},
			"token": schema.StringAttribute{
				Description: "The token.",
				Computed:    true,
				Sensitive:   true,
			},
			"created_at": schema.StringAttribute{
				Description:   "The UTC timestamp of the created at.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"number_of_uses": schema.Int64Attribute{
				Description: "The number of times that the token is used",
				Computed:    true,
			},
			"is_revoked": schema.BoolAttribute{
				Description: "A flag indicating token has been revoked",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *IdentityTokenAuthTokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create creates the resource and sets the initial Terraform state.
func (r *IdentityTokenAuthTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create identity token auth token",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan IdentityTokenAuthTokenResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createResponse, err := r.client.CreateIdentityTokenAuthToken(infisical.CreateIdentityTokenAuthTokenRequest{
		IdentityID: plan.IdentityID.ValueString(),
		Name:       plan.Name.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating identity token auth token",
			"Couldn't create token auth token in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(createResponse.TokenData.ID)
	plan.Name = types.StringValue(createResponse.TokenData.Name)
	plan.Token = types.StringValue(createResponse.AccessToken)
	plan.TTL = types.Int64Value(createResponse.TokenData.AccessTokenTTL)
	plan.IsRevoked = types.BoolValue(createResponse.TokenData.IsAccessTokenRevoked)
	plan.NumberOfUses = types.Int64Value(createResponse.TokenData.AccessTokenNumUses)
	plan.NumberOfUsesLimit = types.Int64Value(createResponse.TokenData.AccessTokenNumUsesLimit)
	plan.CreatedAt = types.StringValue(createResponse.TokenData.CreatedAt.Format(time.RFC3339))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *IdentityTokenAuthTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read identity token auth token",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state IdentityTokenAuthTokenResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the latest data from the API
	tokenData, err := r.client.GetIdentityTokenAuthToken(infisical.GetIdentityTokenAuthTokenRequest{
		IdentityID: state.IdentityID.ValueString(),
		TokenID:    state.ID.ValueString(),
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError(
				"Error reading identity token auth token",
				"Couldn't read identity token auth token from Infisical, unexpected error: "+err.Error(),
			)
			return
		}
	}

	// Update state with latest data
	state.IsRevoked = types.BoolValue(tokenData.IsAccessTokenRevoked)
	state.NumberOfUses = types.Int64Value(tokenData.AccessTokenNumUses)
	state.NumberOfUsesLimit = types.Int64Value(tokenData.AccessTokenNumUsesLimit)
	state.TTL = types.Int64Value(tokenData.AccessTokenTTL)
	state.Name = types.StringValue(tokenData.Name)
	state.CreatedAt = types.StringValue(tokenData.CreatedAt.Format(time.RFC3339))
	// Note: The token value itself is not returned in the read response for security reasons
	// It's only available during creation

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *IdentityTokenAuthTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update identity token auth token",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan IdentityTokenAuthTokenResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get current state
	var state IdentityTokenAuthTokenResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updatedToken, err := r.client.UpdateIdentityTokenAuthToken(infisical.UpdateIdentityTokenAuthTokenRequest{
		TokenID: state.ID.ValueString(),
		Name:    plan.Name.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating identity token auth token",
			"Couldn't update identity token auth token in Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(updatedToken.ID)
	plan.IdentityID = types.StringValue(updatedToken.IdentityID)
	plan.Name = types.StringValue(updatedToken.Name)
	plan.IsRevoked = types.BoolValue(updatedToken.IsAccessTokenRevoked)
	plan.NumberOfUses = types.Int64Value(updatedToken.AccessTokenNumUses)
	plan.NumberOfUsesLimit = types.Int64Value(updatedToken.AccessTokenNumUsesLimit)
	plan.TTL = types.Int64Value(updatedToken.AccessTokenTTL)
	plan.CreatedAt = types.StringValue(updatedToken.CreatedAt.Format(time.RFC3339))
	plan.Token = state.Token

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *IdentityTokenAuthTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete identity token auth token",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state IdentityTokenAuthTokenResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.RevokeIdentityTokenAuthToken(infisical.RevokeIdentityTokenAuthTokenRequest{
		IdentityID: state.IdentityID.ValueString(),
		TokenID:    state.ID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting identity token auth token",
			"Couldn't revoke identity token auth token in Infisical, unexpected error: "+err.Error(),
		)
		return
	}
}
