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

// NewProjectSecretTagResource is a helper function to simplify the provider implementation.
func NewProjectSecretTagResource() resource.Resource {
	return &projectSecretTagResource{}
}

// projectSecretTagResource is the resource implementation.
type projectSecretTagResource struct {
	client *infisical.Client
}

// projectSecretTagResourceSourceModel describes the data source data model.
type projectSecretTagResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Slug      types.String `tfsdk:"slug"`
	Color     types.String `tfsdk:"color"`
	ProjectID types.String `tfsdk:"project_id"`
}

// Metadata returns the resource type name.
func (r *projectSecretTagResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_tag"
}

// Schema defines the schema for the resource.
func (r *projectSecretTagResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create secret tag & save to Infisical.",
		Attributes: map[string]schema.Attribute{
			"slug": schema.StringAttribute{
				Description: "The slug for the new tag",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name for the new tag",
				Required:    true,
			},
			"color": schema.StringAttribute{
				Description: "Color code for the tag.",
				Required:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "The project id associated with the secret tag",
				Required:    true,
			},
			"id": schema.StringAttribute{
				Description:   "The ID of the role",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *projectSecretTagResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*infisical.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

// Create creates the resource and sets the initial Terraform state.
func (r *projectSecretTagResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client.Config.AuthStrategy != infisical.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY {
		resp.Diagnostics.AddError(
			"Unable to create secret tag",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan projectSecretTagResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	newProjectSecretTag, err := r.client.CreateProjectTag(infisical.CreateProjectTagRequest{
		ProjectID: plan.ProjectID.ValueString(),
		Slug:      plan.Slug.ValueString(),
		Name:      plan.Name.ValueString(),
		Color:     plan.Color.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project secret tag",
			"Couldn't save tag to Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(newProjectSecretTag.Tag.ID)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *projectSecretTagResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client.Config.AuthStrategy != infisical.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY {
		resp.Diagnostics.AddError(
			"Unable to read project tag role",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state projectSecretTagResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the latest data from the API
	secretTag, err := r.client.GetProjectTagByID(infisical.GetProjectTagByIDRequest{
		ProjectID: state.ProjectID.ValueString(),
		TagID:     state.ID.ValueString(),
	})

	if err != nil {
		if err == infisicalclient.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError(
				"Error reading project secret tag",
				"Couldn't read project secret tag from Infiscial, unexpected error: "+err.Error(),
			)
			return
		}
	}

	state.Color = types.StringValue(secretTag.Tag.Color)
	state.Name = types.StringValue(secretTag.Tag.Name)
	state.Slug = types.StringValue(secretTag.Tag.Slug)
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *projectSecretTagResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client.Config.AuthStrategy != infisical.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY {
		resp.Diagnostics.AddError(
			"Unable to update secret tag",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan projectSecretTagResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state projectSecretTagResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdateProjectTag(infisical.UpdateProjectTagRequest{
		ProjectID: plan.ProjectID.ValueString(),
		Slug:      plan.Slug.ValueString(),
		Name:      plan.Name.ValueString(),
		Color:     plan.Color.ValueString(),
		TagID:     plan.ID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating secret tag",
			"Couldn't update secret tag from Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Delete deletes the resource and removes the Terraform state on success.
func (r *projectSecretTagResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	if r.client.Config.AuthStrategy != infisical.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY {
		resp.Diagnostics.AddError(
			"Unable to delete secret tag",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state projectSecretTagResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteProjectTag(infisical.DeleteProjectTagRequest{
		ProjectID: state.ProjectID.ValueString(),
		TagID:     state.ID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting secret tag",
			"Couldn't delete secret tag from Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

}
