package resource

import (
	"context"
	"fmt"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
	infisicalclient "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NewProjectSecretFolderResource is a helper function to simplify the provider implementation.
func NewProjectSecretFolderResource() resource.Resource {
	return &projectSecretFolderResource{}
}

// projectSecretFolderResource is the resource implementation.
type projectSecretFolderResource struct {
	client *infisical.Client
}

// projectSecretFolderResourceSourceModel describes the data source data model.
type projectSecretFolderResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	EnvironmentID   types.String `tfsdk:"environment_id"`
	ProjectID       types.String `tfsdk:"project_id"`
	EnvironmentSlug types.String `tfsdk:"environment_slug"`
	SecretPath      types.String `tfsdk:"folder_path"`
}

// Metadata returns the resource type name.
func (r *projectSecretFolderResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_folder"
}

// Schema defines the schema for the resource.
func (r *projectSecretFolderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create secret folder & save to Infisical.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name for the folder",
				Required:    true,
			},
			"folder_path": schema.StringAttribute{
				Description:   "The path where the folder should be created/updated",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"environment_slug": schema.StringAttribute{
				Description:   "The environment slug of the folder to modify/create",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"project_id": schema.StringAttribute{
				Description:   "The Infisical project ID (Required for Machine Identity auth, and service tokens with multiple scopes)",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"id": schema.StringAttribute{
				Description:   "The ID of the folder",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"environment_id": schema.StringAttribute{
				Description: "The ID of the environment",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *projectSecretFolderResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *projectSecretFolderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create secret folder",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan projectSecretFolderResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	newProjectSecretFolder, err := r.client.CreateSecretFolder(infisical.CreateSecretFolderRequest{
		Name:        plan.Name.ValueString(),
		ProjectID:   plan.ProjectID.ValueString(),
		Environment: plan.EnvironmentSlug.ValueString(),
		SecretPath:  plan.SecretPath.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project secret folder",
			"Couldn't save folder to Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(newProjectSecretFolder.Folder.ID)
	plan.EnvironmentID = types.StringValue(newProjectSecretFolder.Folder.EnvID)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *projectSecretFolderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read project folder",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state projectSecretFolderResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the latest data from the API
	secretFolder, err := r.client.GetSecretFolderByID(infisical.GetSecretFolderByIDRequest{
		ID: state.ID.ValueString(),
	})

	if err != nil {
		if err == infisicalclient.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError(
				"Error fetching folders from your project",
				"Couldn't read project secret folder from Infiscial, unexpected error: "+err.Error(),
			)
			return
		}
	}

	state.EnvironmentID = types.StringValue(secretFolder.Folder.EnvID)
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *projectSecretFolderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update secret folder",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan projectSecretFolderResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state projectSecretFolderResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updatedFolder, err := r.client.UpdateSecretFolder(infisical.UpdateSecretFolderRequest{
		ProjectID:   plan.ProjectID.ValueString(),
		Name:        plan.Name.ValueString(),
		ID:          plan.ID.ValueString(),
		Environment: plan.EnvironmentSlug.ValueString(),
		SecretPath:  plan.SecretPath.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating secret folder",
			"Couldn't update secret folder from Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

	plan.EnvironmentID = types.StringValue(updatedFolder.Folder.EnvID)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Delete deletes the resource and removes the Terraform state on success.
func (r *projectSecretFolderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete secret folder",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state projectSecretFolderResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteSecretFolder(infisical.DeleteSecretFolderRequest{
		ID:          state.ID.ValueString(),
		ProjectID:   state.ProjectID.ValueString(),
		Environment: state.EnvironmentSlug.ValueString(),
		SecretPath:  state.SecretPath.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting secret folder",
			"Couldn't delete secret folder from Infiscial, unexpected error: "+err.Error(),
		)
		return
	}

}

func (r *projectSecretFolderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

	// secret_folder_id

	folder, err := r.client.GetSecretFolderByID(infisical.GetSecretFolderByIDRequest{
		ID: req.ID,
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.Diagnostics.AddError(
				"Secrets folder not found",
				"The secrets folder with the given ID was not found",
			)
		} else {
			resp.Diagnostics.AddError(
				"Error fetching secrets folder",
				"Couldn't fetch secrets folder from Infiscial, unexpected error: "+err.Error(),
			)
		}
		return
	}

	// Remove leading and trailing slashes.
	trimmedPath := strings.Trim(folder.Folder.Path, "/")
	var name string
	parentFolderPath := "/"

	if trimmedPath != "" {
		// Split the path and get the last element as the name.
		pathParts := strings.Split(trimmedPath, "/")
		name = pathParts[len(pathParts)-1]
		if len(pathParts) > 1 {
			// At the moment, the folder path returned by GetByID includes the folder being queried.
			// We want to be consistent with the create conventions, so we remove the last part
			parentFolderPath += strings.Join(pathParts[:len(pathParts)-1], "/")
		}
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("folder_path"), parentFolderPath)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), folder.Folder.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_slug"), folder.Folder.Environment.EnvSlug)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), folder.Folder.ProjectID)...)
}
