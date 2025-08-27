package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// NewIdentityUniversalAuthClientSecretResource is a helper function to simplify the provider implementation.
func NewIdentityUniversalAuthClientSecretResource() resource.Resource {
	return &IdentityUniversalAuthClientSecretResource{}
}

// IdentityUniversalAuthClientSecretResource is the resource implementation.
type IdentityUniversalAuthClientSecretResource struct {
	client *infisical.Client
}

// IdentityUniversalAuthClientSecretResourceSourceModel describes the data source data model.
type IdentityUniversalAuthClientSecretResourceModel struct {
	ID                types.String `tfsdk:"id"`
	IdentityID        types.String `tfsdk:"identity_id"`
	Description       types.String `tfsdk:"description"`
	NumberOfUsesLimit types.Int64  `tfsdk:"number_of_uses_limit"`
	NumberOfUses      types.Int64  `tfsdk:"number_of_uses"`
	TTL               types.Int64  `tfsdk:"ttl"`
	ClientID          types.String `tfsdk:"client_id"`
	ClientSecret      types.String `tfsdk:"client_secret"`
	CreatedAt         types.String `tfsdk:"created_at"`
	IsRevoked         types.Bool   `tfsdk:"is_revoked"`
}

// Metadata returns the resource type name.
func (r *IdentityUniversalAuthClientSecretResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_universal_auth_client_secret"
}

// Schema defines the schema for the resource.
func (r *IdentityUniversalAuthClientSecretResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage identity universal auth client secret in Infisical.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the universal auth client secret",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"identity_id": schema.StringAttribute{
				Description:   "The ID of the identity to create a client secret for",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"description": schema.StringAttribute{
				Description: "The description of the client secret.",
				Optional:    true,
			},
			"number_of_uses_limit": schema.Int64Attribute{
				Description: "The maximum number of times that the client secret can be used; a value of 0 implies infinite number of uses. Default: 0",
				Optional:    true,
				Computed:    true,
			},
			"ttl": schema.Int64Attribute{
				Description: "The lifetime for the client secret in seconds. Default: 0 - not expiring",
				Optional:    true,
				Computed:    true,
			},
			"client_id": schema.StringAttribute{
				Description: "The client ID of the secret.",
				Computed:    true,
			},
			"client_secret": schema.StringAttribute{
				Description: "The client secret.",
				Computed:    true,
				Sensitive:   true,
			},
			"created_at": schema.StringAttribute{
				Description:   "The UTC timestamp of the created at.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			"number_of_uses": schema.Int64Attribute{
				Description: "The number of times that the client secret is used",
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
func (r *IdentityUniversalAuthClientSecretResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *IdentityUniversalAuthClientSecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create identity universal auth client secret",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan IdentityUniversalAuthClientSecretResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	universalAuth, err := r.client.GetIdentityUniversalAuth(infisical.GetIdentityUniversalAuthRequest{
		IdentityID: plan.IdentityID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating identity universal auth client secret",
			"Couldn't save universal auth client secret to Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	newIdentityUniversalAuthClientSecret, err := r.client.CreateIdentityUniversalAuthClientSecret(infisical.CreateIdentityUniversalAuthClientSecretRequest{
		IdentityID:   plan.IdentityID.ValueString(),
		Description:  plan.Description.ValueString(),
		NumUsesLimit: plan.NumberOfUsesLimit.ValueInt64(),
		TTL:          plan.TTL.ValueInt64(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating identity universal auth client secret",
			"Couldn't save universal auth client secret to Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(newIdentityUniversalAuthClientSecret.ClientSecretData.ID)
	plan.ClientID = types.StringValue(universalAuth.ClientID)
	plan.TTL = types.Int64Value(newIdentityUniversalAuthClientSecret.ClientSecretData.ClientSecretTTL)
	plan.ClientSecret = types.StringValue(newIdentityUniversalAuthClientSecret.ClientSecret)
	plan.IsRevoked = types.BoolValue(newIdentityUniversalAuthClientSecret.ClientSecretData.IsClientSecretRevoked)
	plan.NumberOfUses = types.Int64Value(newIdentityUniversalAuthClientSecret.ClientSecretData.ClientSecretNumUses)
	plan.NumberOfUsesLimit = types.Int64Value(newIdentityUniversalAuthClientSecret.ClientSecretData.ClientSecretNumUsesLimit)
	plan.CreatedAt = types.StringValue(newIdentityUniversalAuthClientSecret.ClientSecretData.CreatedAt)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *IdentityUniversalAuthClientSecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read identity universal auth client secret role",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state IdentityUniversalAuthClientSecretResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the latest data from the API
	identityUniversalAuthClientSecretData, err := r.client.GetIdentityUniversalAuthClientSecret(infisical.GetIdentityUniversalAuthClientSecretRequest{
		IdentityID:     state.IdentityID.ValueString(),
		ClientSecretID: state.ID.ValueString(),
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError(
				"Error reading identity universal auth client secret",
				"Couldn't read identity universal auth client secret from Infiscial, unexpected error: "+err.Error(),
			)
			return
		}
	}

	state.IsRevoked = types.BoolValue(identityUniversalAuthClientSecretData.IsClientSecretRevoked)
	state.NumberOfUses = types.Int64Value(identityUniversalAuthClientSecretData.ClientSecretNumUses)
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *IdentityUniversalAuthClientSecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update identity universal auth client secret",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	tflog.Error(ctx, "Client secrets are immutable and cannot be modified once they are generated.")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *IdentityUniversalAuthClientSecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete identity universal auth client secret",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state IdentityUniversalAuthClientSecretResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.RevokeIdentityUniversalAuthClientSecret(infisical.RevokeIdentityUniversalAuthClientSecretRequest{
		IdentityID:     state.IdentityID.ValueString(),
		ClientSecretID: state.ID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting identity universal auth client secret",
			"Couldn't delete identity universal auth client secret from Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

}
