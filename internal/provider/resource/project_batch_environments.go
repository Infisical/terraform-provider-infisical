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
)

// NewProjectEnvironmentResource is a helper function to simplify the provider implementation.
func NewProjectBatchEnvironmentsResource() resource.Resource {
	return &projectBatchEnvironmentsResource{}
}

// projectEnvironmentResource is the resource implementation.
type projectBatchEnvironmentsResource struct {
	client *infisical.Client
}

type projectBatchEnvironmentsResourceModel struct {
	ProjectID    types.String `tfsdk:"project_id"`
	Environments []struct {
		ID   types.String `tfsdk:"id"`
		Slug types.String `tfsdk:"slug"`
		Name types.String `tfsdk:"name"`
	} `tfsdk:"environments"`
}

// Metadata returns the resource type name.
func (r *projectBatchEnvironmentsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_batch_project_environment"
}

// Schema defines the schema for the resource.
func (r *projectBatchEnvironmentsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create project environment",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description:   "The Infisical project ID (Required for Machine Identity auth, and service tokens with multiple scopes)",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"environments": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
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
					},
				},
				Description: "The environments to create",
				Required:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *projectBatchEnvironmentsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *projectBatchEnvironmentsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create project environment",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan projectBatchEnvironmentsResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var environments []infisical.BasicEnvironment
	for _, environment := range plan.Environments {

		environments = append(environments, infisical.BasicEnvironment{
			Name: environment.Name.ValueString(),
			Slug: environment.Slug.ValueString(),
		})

	}

	newProjectEnvironments, err := r.client.CreateBatchProjectEnvironments(infisical.CreateBatchProjectEnvironmentsRequest{
		Environments: environments,
		ProjectID:    plan.ProjectID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project environment",
			"Couldn't save project environment, unexpected error: "+err.Error(),
		)
		return
	}

	newPlan := projectBatchEnvironmentsResourceModel{}
	for _, newProjectEnvironment := range newProjectEnvironments.Environments {

		newPlan.Environments = append(newPlan.Environments, struct {
			ID   types.String `tfsdk:"id"`
			Slug types.String `tfsdk:"slug"`
			Name types.String `tfsdk:"name"`
		}{
			ID:   types.StringValue(newProjectEnvironment.ID),
			Slug: types.StringValue(newProjectEnvironment.Slug),
			Name: types.StringValue(newProjectEnvironment.Name),
		})
	}

	plan.ProjectID = types.StringValue(plan.ProjectID.ValueString())
	plan.Environments = newPlan.Environments

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *projectBatchEnvironmentsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete project environment",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state projectBatchEnvironmentsResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	environmentIds := []string{}
	for _, environment := range state.Environments {
		environmentIds = append(environmentIds, environment.ID.ValueString())
	}

	_, err := r.client.DeleteBatchProjectEnvironments(infisical.DeleteBatchProjectEnvironmentsRequest{
		EnvironmentIds: environmentIds,
		ProjectID:      state.ProjectID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting project environments",
			"Couldn't delete project environments from Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *projectBatchEnvironmentsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read project environment",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state projectBatchEnvironmentsResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var environmentIds []string
	for _, environment := range state.Environments {
		environmentIds = append(environmentIds, environment.ID.ValueString())
	}

	projectEnvironment, err := r.client.GetBatchProjectEnvironments(infisical.GetBatchProjectEnvironmentsRequest{
		EnvironmentIds: environmentIds,
		ProjectID:      state.ProjectID.ValueString(),
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

	state.Environments = []struct {
		ID   types.String `tfsdk:"id"`
		Slug types.String `tfsdk:"slug"`
		Name types.String `tfsdk:"name"`
	}{}

	for _, environment := range projectEnvironment.Environments {
		state.Environments = append(state.Environments, struct {
			ID   types.String `tfsdk:"id"`
			Slug types.String `tfsdk:"slug"`
			Name types.String `tfsdk:"name"`
		}{
			ID:   types.StringValue(environment.ID),
			Slug: types.StringValue(environment.Slug),
			Name: types.StringValue(environment.Name),
		})
	}

	state.ProjectID = types.StringValue(state.ProjectID.ValueString())

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *projectBatchEnvironmentsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update project environment",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan projectBatchEnvironmentsResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state projectBatchEnvironmentsResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var environmentsToUpdate []infisical.BasicEnvironmentUpdate
	var environmentsToDelete []string

	// Create a map of environment IDs in the plan for quick lookup
	planEnvs := make(map[string]struct{})
	for _, env := range plan.Environments {
		planEnvs[env.ID.ValueString()] = struct{}{}
	}

	// Check for environments to delete
	for _, stateEnv := range state.Environments {
		if _, exists := planEnvs[stateEnv.ID.ValueString()]; !exists {
			environmentsToDelete = append(environmentsToDelete, stateEnv.ID.ValueString())
		}
	}

	for _, environment := range plan.Environments {
		if environment.Name.ValueString() == "" {
			resp.Diagnostics.AddError(
				"Error updating project environment",
				"Name is required",
			)
			return
		}
		if environment.Slug.ValueString() == "" {
			resp.Diagnostics.AddError(
				"Error updating project environment",
				"Slug is required",
			)
			return
		}
	}

	for _, environment := range plan.Environments {

		updateEnv := infisical.BasicEnvironmentUpdate{
			ID: environment.ID.ValueString(),
		}

		if environment.Name.ValueString() != state.Environments[0].Name.ValueString() {
			updateEnv.Name = environment.Name.ValueString()
		}

		if environment.Slug.ValueString() != state.Environments[0].Slug.ValueString() {
			updateEnv.Slug = environment.Slug.ValueString()
		}

		environmentsToUpdate = append(environmentsToUpdate, updateEnv)
	}

	updatedEnvironment, err := r.client.UpdateBatchProjectEnvironments(infisical.UpdateBatchProjectEnvironmentsRequest{
		Environments: environmentsToUpdate,
		ProjectID:    plan.ProjectID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project environment",
			"Couldn't update project environment from Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	newPlan := projectBatchEnvironmentsResourceModel{}
	for _, updatedEnvironment := range updatedEnvironment.Environments {
		newPlan.Environments = append(plan.Environments, struct {
			ID   types.String `tfsdk:"id"`
			Slug types.String `tfsdk:"slug"`
			Name types.String `tfsdk:"name"`
		}{
			ID:   types.StringValue(updatedEnvironment.ID),
			Slug: types.StringValue(updatedEnvironment.Slug),
			Name: types.StringValue(updatedEnvironment.Name),
		})
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
