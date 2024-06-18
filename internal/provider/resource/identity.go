package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"
	infisicalclient "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NewIdentityResource is a helper function to simplify the provider implementation.
func NewIdentityResource() resource.Resource {
	return &IdentityResource{}
}

// IdentityResource is the resource implementation.
type IdentityResource struct {
	client *infisical.Client
}

// IdentityResourceSourceModel describes the data source data model.
type IdentityResourceModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	AuthMode types.String `tfsdk:"auth_mode"`
	Role     types.String `tfsdk:"role"`
	OrgID    types.String `tfsdk:"org_id"`
}

// Metadata returns the resource type name.
func (r *IdentityResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity"
}

// Schema defines the schema for the resource.
func (r *IdentityResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage identity in Infisical.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name for the identity",
				Required:    true,
			},
			"org_id": schema.StringAttribute{
				Description:   "The ID of the organization for the identity",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"role": schema.StringAttribute{
				Description: "The role for the identity",
				Required:    true,
			},
			"id": schema.StringAttribute{
				Description:   "The ID of the role",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			"auth_mode": schema.StringAttribute{
				Description: "The authentication type of the identity",
				Computed:    true,
				Optional:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *IdentityResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *IdentityResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client.Config.AuthStrategy != infisical.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY {
		resp.Diagnostics.AddError(
			"Unable to create identity",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan IdentityResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	newIdentity, err := r.client.CreateIdentity(infisical.CreateIdentityRequest{
		OrgID: plan.OrgID.ValueString(),
		Name:  plan.Name.ValueString(),
		Role:  plan.Role.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating identity",
			"Couldn't save tag to Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(newIdentity.Identity.ID)
	if newIdentity.Identity.AuthMethod != "" {
		plan.AuthMode = types.StringValue(newIdentity.Identity.AuthMethod)
	} else {
		plan.AuthMode = types.StringNull()
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *IdentityResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client.Config.AuthStrategy != infisical.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY {
		resp.Diagnostics.AddError(
			"Unable to read identity role",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state IdentityResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the latest data from the API
	orgIdentity, err := r.client.GetIdentity(infisical.GetIdentityRequest{
		IdentityID: state.ID.ValueString(),
	})

	if err != nil {
		if err == infisicalclient.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError(
				"Error reading identity",
				"Couldn't read identity from Infiscial, unexpected error: "+err.Error(),
			)
			return
		}
	}

	state.Name = types.StringValue(orgIdentity.Identity.Name)
	if orgIdentity.Identity.AuthMethod != "" {
		state.AuthMode = types.StringValue(orgIdentity.Identity.AuthMethod)
	} else {
		state.AuthMode = types.StringNull()
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *IdentityResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client.Config.AuthStrategy != infisical.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY {
		resp.Diagnostics.AddError(
			"Unable to update identity",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan IdentityResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state IdentityResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgIdentity, err := r.client.UpdateIdentity(infisical.UpdateIdentityRequest{
		IdentityID: state.ID.ValueString(),
		Name:       plan.Name.ValueString(),
		Role:       plan.Role.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating identity",
			"Couldn't update identity from Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	if orgIdentity.Identity.AuthMethod != "" {
		plan.AuthMode = types.StringValue(orgIdentity.Identity.AuthMethod)
	} else {
		plan.AuthMode = types.StringNull()
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Delete deletes the resource and removes the Terraform state on success.
func (r *IdentityResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	if r.client.Config.AuthStrategy != infisical.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY {
		resp.Diagnostics.AddError(
			"Unable to delete identity",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state IdentityResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteIdentity(infisical.DeleteIdentityRequest{
		IdentityID: state.ID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting identity",
			"Couldn't delete identity from Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

}
