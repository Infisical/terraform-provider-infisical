package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"
	infisicalclient "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SecretSyncBaseResource is the resource implementation.
type SecretSyncBaseResource struct {
	App                                    infisicalclient.SecretSyncApp // used for identifying secret sync route
	ResourceTypeName                       string                        // terraform resource name suffix
	SyncName                               string                        // complete descriptive name of the secret sync
	AppConnection                          infisicalclient.AppConnectionApp
	client                                 *infisical.Client
	DestinationConfigAttributes            map[string]schema.Attribute
	ReadDestinationConfigForCreateFromPlan func(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics)
	ReadDestinationConfigForUpdateFromPlan func(ctx context.Context, plan SecretSyncBaseResourceModel, state SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics)
	ReadDestinationConfigFromApi           func(ctx context.Context, secretSync infisicalclient.SecretSync) (types.Object, diag.Diagnostics)

	SyncOptionsAttributes            map[string]schema.Attribute
	ReadSyncOptionsForCreateFromPlan func(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics)
	ReadSyncOptionsForUpdateFromPlan func(ctx context.Context, plan SecretSyncBaseResourceModel, state SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics)
	ReadSyncOptionsFromApi           func(ctx context.Context, secretSync infisicalclient.SecretSync) (types.Object, diag.Diagnostics)
}

type SecretSyncBaseResourceModel struct {
	ID                types.String `tfsdk:"id"`
	ConnectionID      types.String `tfsdk:"connection_id"`
	Name              types.String `tfsdk:"name"`
	ProjectID         types.String `tfsdk:"project_id"`
	Description       types.String `tfsdk:"description"`
	Environment       types.String `tfsdk:"environment"`
	SecretPath        types.String `tfsdk:"secret_path"`
	SyncOptions       types.Object `tfsdk:"sync_options"`
	AutoSyncEnabled   types.Bool   `tfsdk:"auto_sync_enabled"`
	DestinationConfig types.Object `tfsdk:"destination_config"`
}

// Metadata returns the resource type name.
func (r *SecretSyncBaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.ResourceTypeName
}

func (r *SecretSyncBaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("Create and manage %s secret syncs", r.SyncName),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   fmt.Sprintf("The ID of the %s secret sync", r.SyncName),
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"connection_id": schema.StringAttribute{
				Required:    true,
				Description: fmt.Sprintf("The ID of the %s Connection to use for syncing.", r.AppConnection),
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: fmt.Sprintf("The name of the %s sync to create. Must be slug-friendly.", r.SyncName),
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
			"description": schema.StringAttribute{
				Optional:    true,
				Description: fmt.Sprintf("An optional description for the %s sync.", r.SyncName),
			},
			"auto_sync_enabled": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether secrets should be automatically synced when changes occur at the source location or not.",
				Default:     booldefault.StaticBool(true),
				Computed:    true,
			},
			"sync_options": schema.SingleNestedAttribute{
				Required:    true,
				Description: "Parameters to modify how secrets are synced.",
				Attributes:  r.SyncOptionsAttributes,
			},
			"destination_config": schema.SingleNestedAttribute{
				Required:    true,
				Description: "The destination configuration for the secret sync.",
				Attributes:  r.DestinationConfigAttributes,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *SecretSyncBaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *SecretSyncBaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to create secret sync",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan SecretSyncBaseResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	syncOptions, diags := r.ReadSyncOptionsForCreateFromPlan(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	initialSyncBehavior, ok := syncOptions["initialSyncBehavior"].(string)
	if !ok {
		initialSyncBehavior = ""
	}

	if initialSyncBehavior == "" {
		resp.Diagnostics.AddError(
			"Unable to create secret sync",
			"Failed to parse initial_sync_behavior field",
		)
		return
	}

	switch infisical.SecretSyncBehavior(initialSyncBehavior) {
	case infisical.SecretSyncBehaviorOverwriteDestination, infisical.SecretSyncBehaviorPrioritizeDestination, infisical.SecretSyncBehaviorPrioritizeSource:
		break
	default:
		resp.Diagnostics.AddError(
			"Unable to create secret sync",
			fmt.Sprintf("Invalid value for initial_sync_behavior field. Possible values are: %s, %s, %s",
				infisical.SecretSyncBehaviorOverwriteDestination, infisical.SecretSyncBehaviorPrioritizeDestination,
				infisical.SecretSyncBehaviorPrioritizeSource),
		)
		return
	}

	destinationConfigMap, diags := r.ReadDestinationConfigForCreateFromPlan(ctx, plan)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	secretSync, err := r.client.CreateSecretSync(infisicalclient.CreateSecretSyncRequest{
		App:               r.App,
		Name:              plan.Name.ValueString(),
		Description:       plan.Description.ValueString(),
		ProjectID:         plan.ProjectID.ValueString(),
		ConnectionID:      plan.ConnectionID.ValueString(),
		Environment:       plan.Environment.ValueString(),
		SecretPath:        plan.SecretPath.ValueString(),
		AutoSyncEnabled:   plan.AutoSyncEnabled.ValueBool(),
		SyncOptions:       syncOptions,
		DestinationConfig: destinationConfigMap,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating secret sync",
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
func (r *SecretSyncBaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to read secret sync",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Get current state
	var state SecretSyncBaseResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	secretSync, err := r.client.GetSecretSyncById(infisicalclient.GetSecretSyncByIdRequest{
		App: r.App,
		ID:  state.ID.ValueString(),
	})

	if err != nil {
		if err == infisical.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		} else {
			resp.Diagnostics.AddError(
				"Error reading secret sync",
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
	state.AutoSyncEnabled = types.BoolValue(secretSync.AutoSyncEnabled)

	if !(state.Description.IsNull() && secretSync.Description == "") {
		state.Description = types.StringValue(secretSync.Description)
	}

	state.SyncOptions, diags = r.ReadSyncOptionsFromApi(ctx, secretSync)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.DestinationConfig, diags = r.ReadDestinationConfigFromApi(ctx, secretSync)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *SecretSyncBaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update secret sync",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	// Retrieve values from plan
	var plan SecretSyncBaseResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state SecretSyncBaseResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	secretSyncPlan, diags := r.ReadSyncOptionsForUpdateFromPlan(ctx, plan, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	initialSyncBehavior, ok := secretSyncPlan["initialSyncBehavior"].(string)
	if !ok {
		initialSyncBehavior = ""
	}

	if initialSyncBehavior == "" {
		resp.Diagnostics.AddError(
			"Unable to update secret sync",
			"Failed to parse initial_sync_behavior field",
		)
		return
	}

	switch infisical.SecretSyncBehavior(initialSyncBehavior) {
	case infisical.SecretSyncBehaviorOverwriteDestination, infisical.SecretSyncBehaviorPrioritizeDestination, infisical.SecretSyncBehaviorPrioritizeSource:
		break
	default:
		resp.Diagnostics.AddError(
			"Unable to update secret sync",
			fmt.Sprintf("Invalid value for initial_sync_behavior field. Possible values are: %s, %s, %s",
				infisical.SecretSyncBehaviorOverwriteDestination, infisical.SecretSyncBehaviorPrioritizeDestination,
				infisical.SecretSyncBehaviorPrioritizeSource),
		)
		return
	}

	destinationConfigMap, diags := r.ReadDestinationConfigForUpdateFromPlan(ctx, plan, state)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	_, err := r.client.UpdateSecretSync(infisicalclient.UpdateSecretSyncRequest{
		App:               r.App,
		ID:                state.ID.ValueString(),
		Name:              plan.Name.ValueString(),
		Description:       plan.Description.ValueString(),
		ProjectID:         plan.ProjectID.ValueString(),
		ConnectionID:      plan.ConnectionID.ValueString(),
		Environment:       plan.Environment.ValueString(),
		SecretPath:        plan.SecretPath.ValueString(),
		AutoSyncEnabled:   plan.AutoSyncEnabled.ValueBool(),
		SyncOptions:       secretSyncPlan,
		DestinationConfig: destinationConfigMap,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating secret sync",
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

func (r *SecretSyncBaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete secret sync",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var state SecretSyncBaseResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteSecretSync(infisical.DeleteSecretSyncRequest{
		App: r.App,
		ID:  state.ID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting secret sync",
			"Couldn't delete secret sync from Infisical, unexpected error: "+err.Error(),
		)
	}
}
