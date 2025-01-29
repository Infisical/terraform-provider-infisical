package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"
	infisicalclient "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const gcpSecretManagerScopeGlobal = "global"

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource = &SecretSyncGcpSecretManagerResource{}
)

// NewSecretSyncGcpSecretManagerResource is a helper function to simplify the provider implementation.
func NewSecretSyncGcpSecretManagerResource() resource.Resource {
	return &SecretSyncGcpSecretManagerResource{}
}

// SecretSyncGcpSecretManagerResource is the resource implementation.
type SecretSyncGcpSecretManagerResource struct {
	client *infisical.Client
}

// SecretSyncGcpSecretManagerResourceModel describes the data source data model.
type SecretSyncGcpSecretManagerResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	ConnectionID        types.String `tfsdk:"connection_id"`
	Name                types.String `tfsdk:"name"`
	ProjectID           types.String `tfsdk:"project_id"`
	Description         types.String `tfsdk:"description"`
	Environment         types.String `tfsdk:"environment"`
	SecretPath          types.String `tfsdk:"secret_path"`
	InitialSyncBehavior types.String `tfsdk:"initial_sync_behavior"`
	AutoSyncEnabled     types.Bool   `tfsdk:"auto_sync_enabled"`
	GcpProjectID        types.String `tfsdk:"gcp_project_id"`
	Scope               types.String `tfsdk:"scope"`
}

// Metadata returns the resource type name.
func (r *SecretSyncGcpSecretManagerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_sync_gcp_secret_manager"
}

// Schema defines the schema for the resource.
func (r *SecretSyncGcpSecretManagerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manage GCP Secret Manager secret syncs",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The ID of the GCP Secret Manager secret sync",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"connection_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the GCP Connection to use for syncing.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the GCP Secret Manager sync to create. Must be slug-friendly.",
			},
			"project_id": schema.StringAttribute{
				Required:      true,
				Description:   "The ID of the Infisical project to create the sync in.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"environment": schema.StringAttribute{
				Required:    true,
				Description: "The slug of the project environment to sync secrets from.",
			},
			"secret_path": schema.StringAttribute{
				Required:    true,
				Description: "The folder path to sync secrets from.",
			},
			"gcp_project_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the GCP project to sync with",
			},
			"scope": schema.StringAttribute{
				Optional:    true,
				Description: "The scope of the sync with GCP Secret Manager. Supported options: global",
				Default:     stringdefault.StaticString("global"),
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "An optional description for the GCP Secret Manager sync.",
			},
			"initial_sync_behavior": schema.StringAttribute{
				Required:    true,
				Description: "Specify how Infisical should resolve the initial sync to the GCP Secret Manager destination. Supported options: overwrite-destination, import-prioritize-source, import-prioritize-destination",
			},
			"auto_sync_enabled": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether secrets should be automatically synced when changes occur at the source location or not.",
				Default:     booldefault.StaticBool(true),
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *SecretSyncGcpSecretManagerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*infisical.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Source Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

// Create creates the resource and sets the initial Terraform state.
func (r *SecretSyncGcpSecretManagerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create GCP Secret Manager secret sync",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan SecretSyncGcpSecretManagerResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Scope.ValueString() != gcpSecretManagerScopeGlobal {
		resp.Diagnostics.AddError(
			"Unable to create GCP secret manager secret sync",
			"Invalid value for scope field. Possible values are: global",
		)
		return
	}

	switch infisical.SecretSyncBehavior(plan.InitialSyncBehavior.ValueString()) {
	case infisical.SecretSyncBehaviorOverwriteDestination, infisical.SecretSyncBehaviorPrioritizeDestination, infisical.SecretSyncBehaviorPrioritizeSource:
		break
	default:
		resp.Diagnostics.AddError(
			"Unable to create GCP secret manager secret sync",
			fmt.Sprintf("Invalid value for initial_sync_behavior field. Possible values are: %s, %s, %s",
				infisical.SecretSyncBehaviorOverwriteDestination, infisical.SecretSyncBehaviorPrioritizeDestination,
				infisical.SecretSyncBehaviorPrioritizeSource),
		)
		return
	}

	destinationConfigMap := map[string]interface{}{}
	destinationConfigMap["scope"] = plan.Scope.ValueString()
	destinationConfigMap["projectId"] = plan.GcpProjectID.ValueString()

	secretSync, err := r.client.CreateSecretSync(infisicalclient.CreateSecretSyncRequest{
		App:             infisicalclient.SecretSyncAppGCPSecretManager,
		Name:            plan.Name.ValueString(),
		Description:     plan.Description.ValueString(),
		ProjectID:       plan.ProjectID.ValueString(),
		ConnectionID:    plan.ConnectionID.ValueString(),
		Environment:     plan.Environment.ValueString(),
		SecretPath:      plan.SecretPath.ValueString(),
		AutoSyncEnabled: plan.AutoSyncEnabled.ValueBool(),
		SyncOptions: infisicalclient.SecretSyncOptions{
			InitialSyncBehavior: plan.InitialSyncBehavior.ValueString(),
		},
		DestinationConfig: destinationConfigMap,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating GCP Secret Manager secret sync",
			"Couldn't create secret sync, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(secretSync.ID)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *SecretSyncGcpSecretManagerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read GCP Secret Manager secret sync",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state SecretSyncGcpSecretManagerResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	secretSync, err := r.client.GetSecretSyncById(infisicalclient.GetSecretSyncByIdRequest{
		App: infisical.SecretSyncAppGCPSecretManager,
		ID:  state.ID.ValueString(),
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError(
				"Error reading GCP Secret Manager secret sync",
				"Couldn't read secret sync, unexpected error: "+err.Error(),
			)
			return
		}
	}

	state.ConnectionID = types.StringValue(secretSync.Connection.ConnectionID)
	state.Name = types.StringValue(secretSync.Name)
	state.ProjectID = types.StringValue(secretSync.ProjectID)
	state.Environment = types.StringValue(secretSync.Environment.Slug)
	state.SecretPath = types.StringValue(secretSync.SecretFolder.Path)
	state.InitialSyncBehavior = types.StringValue(secretSync.SyncOptions.InitialSyncBehavior)
	state.AutoSyncEnabled = types.BoolValue(secretSync.AutoSyncEnabled)

	if !(state.Description.IsNull() && secretSync.Description == "") {
		state.Description = types.StringValue(secretSync.Description)
	}

	if value, ok := secretSync.DestinationConfig["projectId"].(string); ok {
		state.GcpProjectID = types.StringValue(value)
	}

	if value, ok := secretSync.DestinationConfig["scope"].(string); ok {
		state.Scope = types.StringValue(value)
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *SecretSyncGcpSecretManagerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update GCP Secret Manager secret sync",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan SecretSyncGcpSecretManagerResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state SecretSyncGcpSecretManagerResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Scope.ValueString() != gcpSecretManagerScopeGlobal {
		resp.Diagnostics.AddError(
			"Unable to update GCP secret manager secret sync",
			"Invalid value for scope field. Possible values are: global",
		)
		return
	}

	switch infisical.SecretSyncBehavior(plan.InitialSyncBehavior.ValueString()) {
	case infisical.SecretSyncBehaviorOverwriteDestination, infisical.SecretSyncBehaviorPrioritizeDestination, infisical.SecretSyncBehaviorPrioritizeSource:
		break
	default:
		resp.Diagnostics.AddError(
			"Unable to update GCP secret manager secret sync",
			fmt.Sprintf("Invalid value for initial_sync_behavior field. Possible values are: %s, %s, %s",
				infisical.SecretSyncBehaviorOverwriteDestination, infisical.SecretSyncBehaviorPrioritizeDestination,
				infisical.SecretSyncBehaviorPrioritizeSource),
		)
		return
	}

	destinationConfigMap := map[string]interface{}{}
	destinationConfigMap["scope"] = plan.Scope.ValueString()
	destinationConfigMap["projectId"] = plan.GcpProjectID.ValueString()

	_, err := r.client.UpdateSecretSync(infisicalclient.UpdateSecretSyncRequest{
		App:             infisicalclient.SecretSyncAppGCPSecretManager,
		ID:              state.ID.ValueString(),
		Name:            plan.Name.ValueString(),
		Description:     plan.Description.ValueString(),
		ProjectID:       plan.ProjectID.ValueString(),
		ConnectionID:    plan.ConnectionID.ValueString(),
		Environment:     plan.Environment.ValueString(),
		SecretPath:      plan.SecretPath.ValueString(),
		AutoSyncEnabled: plan.AutoSyncEnabled.ValueBool(),
		SyncOptions: infisicalclient.SecretSyncOptions{
			InitialSyncBehavior: plan.InitialSyncBehavior.ValueString(),
		},
		DestinationConfig: destinationConfigMap,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating GCP Secret Manager secret sync",
			"Couldn't update secret sync, unexpected error: "+err.Error(),
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
func (r *SecretSyncGcpSecretManagerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete GCP Secret Manager secret sync",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state SecretSyncGcpSecretManagerResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteSecretSync(infisical.DeleteSecretSyncRequest{
		App: infisical.SecretSyncAppGCPSecretManager,
		ID:  state.ID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting GCP Secret Manager secret sync",
			"Couldn't delete secret sync from Infisical, unexpected error: "+err.Error(),
		)
	}
}
