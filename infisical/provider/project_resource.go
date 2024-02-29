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
	Slug             types.String `tfsdk:"slug"`
	OrganizationSlug types.String `tfsdk:"organization_slug"`
	Name             types.String `tfsdk:"name"`
	LastUpdated      types.String `tfsdk:"last_updated"`
}

// Metadata returns the resource type name.
func (r *projectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

// Schema defines the schema for the resource.
func (r *projectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create projects & save to Infisical. Only Machine Identity authentication is supported for this data source.",
		Attributes: map[string]schema.Attribute{
			"slug": schema.StringAttribute{
				Description: "The slug of the project",
				Required:    true,
			},
			"organization_slug": schema.StringAttribute{
				Description: "The slug of the organization to which the project belongs",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the project",
				Required:    true,
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

	_, err := r.client.CallCreateProject(infisical.CreateProjectRequest{
		OrganizationSlug: plan.OrganizationSlug.ValueString(),
		ProjectName:      plan.Name.ValueString(),
		Slug:             plan.Slug.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project",
			"Couldn't save project to Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	plan.Slug = types.StringValue(plan.Slug.ValueString())
	plan.OrganizationSlug = types.StringValue(plan.OrganizationSlug.ValueString())
	plan.Name = types.StringValue(plan.Name.ValueString())

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
	project, err := r.client.CallGetProject(infisical.GetProjectRequest{
		Slug: state.Slug.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project",
			"Couldn't read project from Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	if state.Name.ValueString() != project.Name {
		state.Name = types.StringValue(project.Name)
	}

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

	var state projectResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.Slug != plan.Slug {
		resp.Diagnostics.AddError(
			"Unable to update project",
			"Slug cannot be updated",
		)
		return
	}

	if state.OrganizationSlug != plan.OrganizationSlug {
		resp.Diagnostics.AddError(
			"Unable to update project",
			"Organization slug cannot be updated",
		)
		return
	}

	_, err := r.client.CallUpdateProject(infisical.UpdateProjectRequest{
		ProjectName: plan.Name.ValueString(),
		Slug:        plan.Slug.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project",
			"Couldn't update project from Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	plan.Name = types.StringValue(plan.Name.ValueString())

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
