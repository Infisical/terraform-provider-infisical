package resource

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
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
		Description: "Imports the secrets of one folder (the source) into another folder (the destination), so the destination can read them. The source can be in the same project or, with `import_project_id`, in another project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the secret import.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project_id": schema.StringAttribute{
				Description:   "The ID of the project to import secrets into (the destination).",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"environment_slug": schema.StringAttribute{
				Description:   "The slug of the environment to import secrets into (the destination), such as `prod`.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"folder_path": schema.StringAttribute{
				Description:   "The path of the folder to import secrets into (the destination), such as `/` or `/app`.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"import_folder_path": schema.StringAttribute{
				Description:   "The path of the folder to import secrets from (the source), such as `/shared`.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"import_environment_slug": schema.StringAttribute{
				Description:   "The slug of the environment to import secrets from (the source), such as `dev`. It belongs to `import_project_id` when that is set, and to `project_id` otherwise.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"import_project_id": schema.StringAttribute{
				Description:   "The ID of the project to import secrets from (the source). Omit it to import from `project_id`, and don't set it to `project_id`. A cross-project import needs cross-project secret sharing turned on for the organization and a project grant in the source project, and can't use replication. Requires Infisical v0.167.17 or later.",
				Optional:      true,
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"is_replication": schema.BoolAttribute{
				Description:   "When `true`, Infisical copies the source secrets into the destination instead of reading them from the source. If an approval policy applies to the destination, the copies are sent as approval requests. Replication needs a plan that includes secret approvals, and cross-project imports can't use it.",
				Required:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
		},
	}
}

// ValidateConfig catches invalid import_project_id values at plan time, before a replacement deletes the existing import.
func (r *projectSecretImportResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config projectSecretImportResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || config.ImportProjectID.IsNull() || config.ImportProjectID.IsUnknown() {
		return
	}

	// Any import_project_id makes replication invalid: another project is a cross-project import, and project_id
	// itself is rejected below. So this check doesn't need project_id to be known.
	if config.IsReplication.ValueBool() {
		resp.Diagnostics.AddAttributeError(
			path.Root("is_replication"),
			"Invalid is_replication",
			"Infisical doesn't support replication for cross-project imports. Set is_replication to false, or remove import_project_id.",
		)
		return
	}

	// A same-project import reads back with import_project_id null, so setting it to project_id would plan a
	// replacement after terraform import or when the attribute is removed.
	if !config.ProjectID.IsUnknown() && config.ProjectID.ValueString() == config.ImportProjectID.ValueString() {
		resp.Diagnostics.AddAttributeError(
			path.Root("import_project_id"),
			"Invalid import_project_id",
			"import_project_id is the same as project_id. Remove import_project_id to import from the same project.",
		)
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
		ImportFrom: infisical.SecretImportSource{
			SourceProjectID: plan.ImportProjectID.ValueString(),
			Environment:     plan.ImportEnvSlug.ValueString(),
			SecretPath:      plan.ImportPath.ValueString(),
		},
	})

	if err != nil {
		if hint := crossProjectImportSetupHint(err, plan); hint != "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("import_project_id"),
				"Error creating project secret import",
				hint+"\n\nInfisical returned: "+err.Error(),
			)
			return
		}
		resp.Diagnostics.AddError(
			"Error creating project secret import",
			"Couldn't save import to Infisical, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(newProjectSecretImport.SecretImport.ID)

	if !plan.ImportProjectID.IsNull() {
		r.verifyCrossProjectImport(ctx, &plan, newProjectSecretImport.SecretImport.ImportEnvironment.ProjectID, resp)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// verifyCrossProjectImport checks that the server created the import from import_project_id. Infisical before v0.167.17
// leaves importEnv.projectId out of the create response, and before v0.161.12 it ignores sourceProjectId and creates
// a same-project import instead. In both cases the import is deleted again, so no wrong import is left behind.
func (r *projectSecretImportResource) verifyCrossProjectImport(ctx context.Context, plan *projectSecretImportResourceModel, createdSourceProjectID string, resp *resource.CreateResponse) {
	if createdSourceProjectID == plan.ImportProjectID.ValueString() {
		return
	}

	_, err := r.client.DeleteSecretImport(infisical.DeleteSecretImportRequest{
		ID:          plan.ID.ValueString(),
		ProjectID:   plan.ProjectID.ValueString(),
		Environment: plan.EnvironmentSlug.ValueString(),
		SecretPath:  plan.SecretPath.ValueString(),
	})
	if err != nil && !errors.As(err, &notFoundErr) {
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
		resp.Diagnostics.AddError(
			"Error creating project secret import",
			"This Infisical instance does not support import_project_id, so the secret import was not created from "+
				"the source project, and deleting it again failed: "+err.Error()+". Destroy the secret import, then "+
				"either upgrade to Infisical v0.167.17 or later or remove import_project_id from the configuration.",
		)
		return
	}

	resp.Diagnostics.AddError(
		"Error creating project secret import",
		"This Infisical instance does not support import_project_id, so the secret import was not created from the "+
			"source project. The secret import was deleted again. Upgrade to Infisical v0.167.17 or later, or remove "+
			"import_project_id from the configuration.",
	)
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

	// Read by ID: the list route hides cross-project imports while cross-project secret sharing is turned off, and
	// only get by ID returns the source project.
	secretImport, err := r.client.GetSecretImportByID(infisical.GetSecretImportByIDRequest{
		ID: state.ID.ValueString(),
	})

	if err != nil {
		if errors.Is(err, infisical.ErrNotFound) {
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"Error fetching secret imports from your project",
				"Couldn't read project secret import from Infisical, unexpected error: "+err.Error(),
			)
		}
		return
	}

	// Delete finds the import by its destination folder, so a folder renamed outside Terraform must reach state.
	state.EnvironmentSlug = types.StringValue(secretImport.SecretImport.Environment.Slug)
	state.SecretPath = secretPathFromAPI(state.SecretPath, secretImport.SecretImport.SecretPath)
	state.ImportProjectID = importProjectIDFromAPI(state.ImportProjectID, secretImport.SecretImport.ProjectID, secretImport.SecretImport.ImportEnvironment.ProjectID)

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
}

// crossProjectImportSetupHint explains the two 403s that createImport in Infisical core returns when a cross-project
// import isn't set up yet. They need an admin to change a setting, which the error message alone doesn't say.
func crossProjectImportSetupHint(err error, plan projectSecretImportResourceModel) string {
	var apiErr *infisicalerrors.APIError
	if plan.ImportProjectID.IsNull() || !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusForbidden {
		return ""
	}

	switch {
	case strings.Contains(apiErr.ErrorMessage, "Cross-project secret sharing is not enabled"):
		return "Cross-project secret sharing is turned off for this organization. An organization admin can turn it on in " +
			"the organization's Product Settings, in the Secrets Management section. Then apply again."
	case strings.Contains(apiErr.ErrorMessage, "No project grant found"):
		return fmt.Sprintf("Project %s doesn't share folder %q of environment %q with project %s. In the settings of project "+
			"%s, share that folder with project %s in the cross-project sharing section. Then apply again.",
			plan.ImportProjectID.ValueString(), plan.ImportPath.ValueString(), plan.ImportEnvSlug.ValueString(),
			plan.ProjectID.ValueString(), plan.ImportProjectID.ValueString(), plan.ProjectID.ValueString())
	}
	return ""
}

// importProjectIDFromAPI keeps import_project_id null for a same-project import, and keeps the current value when the
// API leaves it out (Infisical before v0.167.17).
func importProjectIDFromAPI(current types.String, projectID, apiSourceProjectID string) types.String {
	if apiSourceProjectID == "" || (current.IsNull() && apiSourceProjectID == projectID) {
		return current
	}
	return types.StringValue(apiSourceProjectID)
}

// secretPathFromAPI keeps the configured path when it names the same folder as the API path. The API stores paths in
// canonical form ("app", "/app/" and "//app" all become "/app"), and folder_path requires replacement, so writing the
// canonical form over a configured "/app/" would replace the import on every apply.
func secretPathFromAPI(current types.String, apiPath string) types.String {
	if !current.IsNull() && canonicalSecretPath(current.ValueString()) == canonicalSecretPath(apiPath) {
		return current
	}
	return types.StringValue(apiPath)
}

// canonicalSecretPath follows canonicalizeSecretPath in Infisical core: folder lookups skip empty segments.
func canonicalSecretPath(secretPath string) string {
	var segments []string
	for _, segment := range strings.Split(secretPath, "/") {
		if segment != "" {
			segments = append(segments, segment)
		}
	}
	return "/" + strings.Join(segments, "/")
}
