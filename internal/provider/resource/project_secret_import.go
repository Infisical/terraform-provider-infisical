package resource

import (
	"context"
	"errors"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var notFoundErr *infisical.NotFoundError

// NewProjectSecretImportResource is a helper function to simplify the provider implementation.
func NewProjectSecretImportResource() resource.Resource {
	return &projectSecretImportResource{}
}

// projectSecretImportResource is the resource implementation.
type projectSecretImportResource struct {
	client *infisical.Client
}

// projectSecretImportResourceModel describes the data source data model.
type projectSecretImportResourceModel struct {
	ID              types.String `tfsdk:"id"`
	ProjectID       types.String `tfsdk:"project_id"`
	EnvironmentSlug types.String `tfsdk:"environment_slug"`
	SecretPath      types.String `tfsdk:"folder_path"`
	ImportPath      types.String `tfsdk:"import_folder_path"`
	ImportEnvSlug   types.String `tfsdk:"import_environment_slug"`
	IsReplication   types.Bool   `tfsdk:"is_replication"`
}

// Metadata returns the resource type name.
func (r *projectSecretImportResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_import"
}

// Schema defines the schema for the resource.
func (r *projectSecretImportResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create secret import & save to Infisical.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the secret import",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project_id": schema.StringAttribute{
				Description:   "The Infisical project ID",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"environment_slug": schema.StringAttribute{
				Description:   "The environment slug of the secret import to modify/create",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"folder_path": schema.StringAttribute{
				Description:   "The path where the secret should be imported",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"import_folder_path": schema.StringAttribute{
				Description:   "The path where the secret should be imported from",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"import_environment_slug": schema.StringAttribute{
				Description:   "The environment slug of the secret import to modify/create",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"is_replication": schema.BoolAttribute{
				Description:   "The is_replication of the secret import to modify/create",
				Required:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *projectSecretImportResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *projectSecretImportResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create secret import",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan projectSecretImportResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	newProjectSecretImport, err := r.client.CreateSecretImport(infisical.CreateSecretImportRequest{
		ProjectID:     plan.ProjectID.ValueString(),
		Environment:   plan.EnvironmentSlug.ValueString(),
		SecretPath:    plan.SecretPath.ValueString(),
		IsReplication: plan.IsReplication.ValueBool(),
		ImportFrom: struct {
			Environment string `json:"environment"`
			SecretPath  string `json:"path"`
		}{
			Environment: plan.ImportEnvSlug.ValueString(),
			SecretPath:  plan.ImportPath.ValueString(),
		},
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project secret import",
			"Couldn't save import to Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(newProjectSecretImport.SecretImport.ID)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *projectSecretImportResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read project secret import",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state projectSecretImportResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the latest data from the API
	_, err := r.client.GetSecretImportByID(infisical.GetSecretImportByIDRequest{
		ID:          state.ID.ValueString(),
		ProjectID:   state.ProjectID.ValueString(),
		Environment: state.EnvironmentSlug.ValueString(),
		SecretPath:  state.SecretPath.ValueString(),
	})

	if err != nil {
		if errors.As(err, &notFoundErr) {
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"Error fetching secret imports from your project",
				"Couldn't read project secret import from Infiscial, unexpected error: "+err.Error(),
			)
		}
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *projectSecretImportResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update secret import",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan projectSecretImportResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdateSecretImport(infisical.UpdateSecretImportRequest{
		ProjectID:   plan.ProjectID.ValueString(),
		ID:          plan.ID.ValueString(),
		Environment: plan.EnvironmentSlug.ValueString(),
		SecretPath:  plan.SecretPath.ValueString(),
	})

	if err != nil {
		if errors.As(err, &notFoundErr) {
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"Error updating secret import",
				"Couldn't update secret import from Infiscial, unexpected error: "+err.Error(),
			)
		}
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *projectSecretImportResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete secret import",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state projectSecretImportResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteSecretImport(infisical.DeleteSecretImportRequest{
		ID:          state.ID.ValueString(),
		ProjectID:   state.ProjectID.ValueString(),
		Environment: state.EnvironmentSlug.ValueString(),
		SecretPath:  state.SecretPath.ValueString(),
	})

	if err != nil {
		if errors.As(err, &notFoundErr) {
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"Error deleting secret import",
				"Couldn't delete secret import from Infiscial, unexpected error: "+err.Error(),
			)
		}
		return
	}

}
