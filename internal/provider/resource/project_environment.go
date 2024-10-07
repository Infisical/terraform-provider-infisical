package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NewProjectEnvironmentResource is a helper function to simplify the provider implementation.
func NewProjectEnvironmentResource() resource.Resource {
	return &projectEnvironmentResource{}
}

// projectEnvironmentResource is the resource implementation.
type projectEnvironmentResource struct {
	client *infisical.Client
}

type projectEnvironmentResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Position  types.Int64  `tfsdk:"position"`
	Slug      types.String `tfsdk:"slug"`
	Name      types.String `tfsdk:"name"`
	ProjectID types.String `tfsdk:"project_id"`
}

// Metadata returns the resource type name.
func (r *projectEnvironmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_environment"
}

// Schema defines the schema for the resource.
func (r *projectEnvironmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create project environment",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the environment",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "The name of the environment",
				Required:    true,
			},
			"slug": schema.StringAttribute{
				Description: "The slug of the environment",
				Required:    true,
			},
			"project_id": schema.StringAttribute{
				Description:   "The Infisical project ID (Required for Machine Identity auth, and service tokens with multiple scopes)",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"position": schema.Int64Attribute{
				Description: "The position of the environment",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *projectEnvironmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *projectEnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create project environment",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan projectEnvironmentResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := infisical.CreateProjectEnvironmentRequest{
		Name:      plan.Name.ValueString(),
		ProjectID: plan.ProjectID.ValueString(),
		Slug:      plan.Slug.ValueString(),
	}

	if !plan.Position.IsNull() {
		if plan.Position.ValueInt64() < 0 {
			resp.Diagnostics.AddError(
				"Invalid position",
				"The position must be a positive integer",
			)
			return
		}

		request.Position = plan.Position.ValueInt64()
	}

	newProjectEnvironment, err := r.client.CreateProjectEnvironment(request)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project environment",
			"Couldn't save project environment, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(newProjectEnvironment.Environment.ID)
	plan.Position = types.Int64Value(newProjectEnvironment.Environment.Position)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *projectEnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete project environment",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state projectEnvironmentResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteProjectEnvironment(infisical.DeleteProjectEnvironmentRequest{
		ID:        state.ID.ValueString(),
		ProjectID: state.ProjectID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting project environment",
			"Couldn't delete project environment from Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *projectEnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read project environment",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state projectEnvironmentResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectEnvironment, err := r.client.GetProjectEnvironmentByID(infisical.GetProjectEnvironmentByIDRequest{
		ID: state.ID.ValueString(),
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError(
				"Error fetching environment from your project",
				"Couldn't read project environment from Infiscial, unexpected error: "+err.Error(),
			)
			return
		}
	}

	state.Name = types.StringValue(projectEnvironment.Environment.Name)
	state.Slug = types.StringValue(projectEnvironment.Environment.Slug)
	state.Position = types.Int64Value(projectEnvironment.Environment.Position)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *projectEnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update project environment",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan projectEnvironmentResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state projectEnvironmentResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updatedEnvironment, err := r.client.UpdateProjectEnvironment(infisical.UpdateProjectEnvironmentRequest{
		ProjectID: plan.ProjectID.ValueString(),
		Name:      plan.Name.ValueString(),
		ID:        plan.ID.ValueString(),
		Slug:      plan.Slug.ValueString(),
		Position:  plan.Position.ValueInt64(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project environment",
			"Couldn't update project environment from Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	plan.Name = types.StringValue(updatedEnvironment.Environment.Name)
	plan.Slug = types.StringValue(updatedEnvironment.Environment.Slug)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

func (r *projectEnvironmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

	projectEnvironment, err := r.client.GetProjectEnvironmentByID(infisical.GetProjectEnvironmentByIDRequest{
		ID: req.ID,
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.Diagnostics.AddError(
				"Project environment not found",
				"The project environment with the given slug was not found",
			)
		} else {
			resp.Diagnostics.AddError(
				"Error fetching project environment",
				"Couldn't fetch project environment from Infiscial, unexpected error: "+err.Error(),
			)
		}
		return
	}

	// Set the attributes in the state
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectEnvironment.Environment.ProjectID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), projectEnvironment.Environment.ID)...)
}
