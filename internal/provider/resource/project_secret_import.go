package resource

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	infisical "terraform-provider-infisical/internal/client"
	infisicalerrors "terraform-provider-infisical/internal/errors"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var notFoundErr *infisical.NotFoundError

var _ resource.ResourceWithValidateConfig = &projectSecretImportResource{}

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
	ImportProjectID types.String `tfsdk:"import_project_id"`
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
			"import_project_id": schema.StringAttribute{
				Description:   "The ID of the project to import secrets from. Omit it to import from `project_id`. A cross-project import needs cross-project secret sharing turned on for the organization and a project grant in the source project, and can't use replication. Requires Infisical v0.161.12 or later.",
				Optional:      true,
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
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

// ValidateConfig catches cross-project replication at plan time, before a replacement deletes the existing import.
func (r *projectSecretImportResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config projectSecretImportResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || !config.IsReplication.ValueBool() || config.ImportProjectID.IsNull() {
		return
	}

	if config.ProjectID.IsUnknown() || config.ImportProjectID.IsUnknown() || config.ProjectID.ValueString() == config.ImportProjectID.ValueString() {
		return
	}

	resp.Diagnostics.AddAttributeError(
		path.Root("is_replication"),
		"Invalid is_replication",
		"Infisical doesn't support replication for cross-project imports. Set is_replication to false, or remove import_project_id.",
	)
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
		ImportFrom: infisical.SecretImportSource{
			SourceProjectID: plan.ImportProjectID.ValueString(),
			Environment:     plan.ImportEnvSlug.ValueString(),
			SecretPath:      plan.ImportPath.ValueString(),
		},
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project secret import",
			"Couldn't save import to Infisical, unexpected error: "+err.Error(),
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

	if state.ID.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	// The list routes hide cross-project imports while cross-project sharing is turned off, so read by ID.
	secretImport, err := r.client.GetSecretImportByID(infisical.GetSecretImportByIDRequest{
		ID: state.ID.ValueString(),
	})
	if err != nil {
		if errors.Is(err, infisical.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading secret import",
			"Couldn't read secret import from Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	state.EnvironmentSlug = types.StringValue(secretImport.SecretImport.Environment.Slug)
	state.SecretPath = types.StringValue(secretImport.SecretImport.SecretPath)

	// The by-ID route leaves out the source project. If the list hides the import, keep the value in state.
	if !state.ImportProjectID.IsNull() {
		withSource, err := r.client.GetSecretImport(infisical.GetSecretImportRequest{
			ID:          secretImport.SecretImport.ID,
			ProjectID:   secretImport.SecretImport.ProjectID,
			Environment: secretImport.SecretImport.Environment.Slug,
			SecretPath:  secretImport.SecretImport.SecretPath,
		})
		switch {
		case err == nil:
			state.ImportProjectID = importProjectIDFromAPI(state.ImportProjectID, secretImport.SecretImport.ProjectID, withSource.SecretImport.ImportEnvironment.ProjectID)
		case !errors.As(err, &notFoundErr):
			resp.Diagnostics.AddError(
				"Error reading secret import",
				"Couldn't read the source project of the secret import from Infisical, unexpected error: "+err.Error(),
			)
			return
		}
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
				"Couldn't update secret import from Infisical, unexpected error: "+err.Error(),
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
				"Couldn't delete secret import from Infisical, unexpected error: "+err.Error(),
			)
		}
		return
	}

}

func (r *projectSecretImportResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	secretImport, err := r.client.GetSecretImportByID(infisical.GetSecretImportByIDRequest{
		ID: req.ID,
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.Diagnostics.AddError(
				"Secret Import not found",
				"The secret import with the given ID was not found",
			)
		} else {
			resp.Diagnostics.AddError(
				"Error fetching secret import",
				"Couldn't fetch secret import from Infisical, unexpected error: "+err.Error(),
			)
		}
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), secretImport.SecretImport.ProjectID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), secretImport.SecretImport.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_slug"), secretImport.SecretImport.Environment.Slug)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("folder_path"), secretImport.SecretImport.SecretPath)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("import_folder_path"), secretImport.SecretImport.ImportPath)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("import_environment_slug"), secretImport.SecretImport.ImportEnvironment.Slug)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("is_replication"), secretImport.SecretImport.IsReplication)...)

	// The by-ID route leaves out the source project.
	importProjectID := types.StringNull()
	withSource, err := r.client.GetSecretImport(infisical.GetSecretImportRequest{
		ID:          secretImport.SecretImport.ID,
		ProjectID:   secretImport.SecretImport.ProjectID,
		Environment: secretImport.SecretImport.Environment.Slug,
		SecretPath:  secretImport.SecretImport.SecretPath,
	})
	var apiErr *infisicalerrors.APIError
	switch {
	case err == nil:
		importProjectID = importProjectIDFromAPI(importProjectID, secretImport.SecretImport.ProjectID, withSource.SecretImport.ImportEnvironment.ProjectID)
	case errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound:
		// Servers without the v2 list route predate cross-project imports.
	case errors.As(err, &notFoundErr):
		resp.Diagnostics.AddError(
			"Error fetching secret import",
			fmt.Sprintf("Couldn't read the source project of secret import %q. Infisical lists cross-project imports only while "+
				"cross-project secret sharing is turned on for the organization. Turn it on under Secrets Management > Product Settings and import again.",
				secretImport.SecretImport.ID),
		)
		return
	default:
		resp.Diagnostics.AddError(
			"Error fetching secret import",
			"Couldn't fetch the source project of the secret import from Infisical, unexpected error: "+err.Error(),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("import_project_id"), importProjectID)...)
}

// importProjectIDFromAPI keeps import_project_id null for a same-project import, and keeps the current value when the API leaves it out.
func importProjectIDFromAPI(current types.String, projectID, apiSourceProjectID string) types.String {
	if apiSourceProjectID == "" || (current.IsNull() && apiSourceProjectID == projectID) {
		return current
	}
	return types.StringValue(apiSourceProjectID)
}
