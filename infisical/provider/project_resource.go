package provider

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/client"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource = &projectResource{}
)

// NewProjectResource is a helper function to simplify the provider implementation.
func NewProjectResource() resource.Resource {
	return &projectResource{}
}

// projectResource is the resource implementation.
type projectResource struct {
	client *infisical.Client
}

// projectResourceSourceModel describes the data source data model.
type projectResourceModel struct {
	Slug               types.String `tfsdk:"slug"`
	OrganizationId     types.String `tfsdk:"organization_id"`
	Name               types.String `tfsdk:"name"`
	AutoCapitalization types.Bool   `tfsdk:"auto_capitalization"`
	ProjectId          types.String `tfsdk:"project_Id"`
	LastUpdated        types.String `tfsdk:"last_updated"`
}

// Metadata returns the resource type name.
func (r *projectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

// Schema defines the schema for the resource.
func (r *projectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create projects & save to Infisical",
		Attributes: map[string]schema.Attribute{
			"slug": schema.StringAttribute{
				Description: "The slug of the project. This is optional when creating a project, but for all other operations it is required",
				Required:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "The organization ID of the project",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the project",
				Required:    true,
			},
			"auto_capitalization": schema.StringAttribute{
				Description: "",
				Required:    true,
				Computed:    false,
			},
			"project_Id": schema.StringAttribute{
				Computed: true,
			},

			"last_updated": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *projectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client.Config.AuthStrategy != infisical.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY {
		resp.Diagnostics.AddError(
			"Unable to create project",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan projectResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	res, err := r.client.CallCreateProject(infisical.CreateProjectRequest{
		OrganizationId: plan.OrganizationId.ValueString(),
		ProjectName:    plan.Name.ValueString(),
		Slug:           plan.Slug.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project",
			"Couldn't save project to Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	plan.AutoCapitalization = types.BoolValue(res.Project.AutoCapitalization)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	plan.Name = types.StringValue(res.Project.Name)
	plan.OrganizationId = types.StringValue(res.Project.OrgID)
	plan.ProjectId = types.StringValue(res.Project.ID)
	plan.Slug = types.StringValue(res.Project.Slug)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client.Config.AuthStrategy != infisical.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY {
		resp.Diagnostics.AddError(
			"Unable to read project",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state projectResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the latest data from the API
	res, err := r.client.CallGetProject(infisical.GetProjectRequest{
		Slug: state.Slug.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project",
			"Couldn't read project from Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	state.AutoCapitalization = types.BoolValue(res.Project.AutoCapitalization)
	state.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	state.Name = types.StringValue(res.Project.Name)
	state.OrganizationId = types.StringValue(res.Project.OrgID)
	state.ProjectId = types.StringValue(res.Project.ID)
	state.Slug = types.StringValue(res.Project.Slug)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Update updates the resource and sets the updated Terraform state on success.
func (r *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client.Config.AuthStrategy != infisical.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY {
		resp.Diagnostics.AddError(
			"Unable to update project",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan projectResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updatedProject, err := r.client.CallUpdateProject(infisical.UpdateProjectRequest{
		ProjectName:        plan.Name.ValueString(),
		AutoCapitalization: plan.AutoCapitalization.ValueBool(),
		Slug:               plan.Slug.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project",
			"Couldn't update project from Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	plan.AutoCapitalization = types.BoolValue(updatedProject.AutoCapitalization)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	plan.Name = types.StringValue(updatedProject.Name)
	plan.OrganizationId = types.StringValue(updatedProject.OrgID)
	plan.ProjectId = types.StringValue(updatedProject.ID)
	plan.Slug = types.StringValue(updatedProject.Slug)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Delete deletes the resource and removes the Terraform state on success.
func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	if r.client.Config.AuthStrategy != infisical.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY {
		resp.Diagnostics.AddError(
			"Unable to delete project",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state projectResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.CallDeleteProject(infisical.DeleteProjectRequest{
		Slug: state.Slug.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting project",
			"Couldn't delete project from Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

}
