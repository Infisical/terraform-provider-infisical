package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SecretSyncBaseResource is the resource implementation.
type SecretSyncBaseResource struct {
	App                                    infisical.SecretSyncApp // used for identifying secret sync route
	ResourceTypeName                       string                  // terraform resource name suffix
	SyncName                               string                  // complete descriptive name of the secret sync
	AppConnection                          infisical.AppConnectionApp
	CanImportSecrets                       bool // whether this sync destination supports importing secrets from the destination
	client                                 *infisical.Client
	DestinationConfigAttributes            map[string]schema.Attribute
	ReadDestinationConfigForCreateFromPlan func(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics)
	ReadDestinationConfigForUpdateFromPlan func(ctx context.Context, plan SecretSyncBaseResourceModel, state SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics)
	ReadDestinationConfigFromApi           func(ctx context.Context, secretSync infisical.SecretSync) (types.Object, diag.Diagnostics)

	SyncOptionsAttributes            map[string]schema.Attribute
	ReadSyncOptionsForCreateFromPlan func(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics)
	ReadSyncOptionsForUpdateFromPlan func(ctx context.Context, plan SecretSyncBaseResourceModel, state SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics)
	ReadSyncOptionsFromApi           func(ctx context.Context, secretSync infisical.SecretSync) (types.Object, diag.Diagnostics)

	pollInterval time.Duration // fixed interval between status reads, for tests; zero means the default backoff
}

var (
	_ resource.ResourceWithImportState    = &SecretSyncBaseResource{}
	_ resource.ResourceWithModifyPlan     = &SecretSyncBaseResource{}
	_ resource.ResourceWithValidateConfig = &SecretSyncBaseResource{}
)

type SecretSyncBaseResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	ConnectionID         types.String `tfsdk:"connection_id"`
	Name                 types.String `tfsdk:"name"`
	ProjectID            types.String `tfsdk:"project_id"`
	Description          types.String `tfsdk:"description"`
	Environment          types.String `tfsdk:"environment"`
	SecretPath           types.String `tfsdk:"secret_path"`
	IncludeAllSubFolders types.Bool   `tfsdk:"include_all_sub_folders"`
	SyncOptions          types.Object `tfsdk:"sync_options"`
	AutoSyncEnabled      types.Bool   `tfsdk:"auto_sync_enabled"`
	DestinationConfig    types.Object `tfsdk:"destination_config"`

	// Provider-only settings. The API doesn't know them, so Read keeps them from state.
	WaitForSyncCompletion types.Bool     `tfsdk:"wait_for_sync_completion"`
	Timeouts              timeouts.Value `tfsdk:"timeouts"`
}

// Metadata returns the resource type name.
func (r *SecretSyncBaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.ResourceTypeName
}

// ImportState imports an existing secret sync by its ID. Read then populates every attribute.
func (r *SecretSyncBaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := uuid.Parse(req.ID); err != nil {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected the %s secret sync ID to be a valid UUID, got: %s", r.SyncName, req.ID),
		)
		return
	}

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *SecretSyncBaseResource) waitForSyncCompletionDescription() string {
	description := "Whether Terraform waits for the sync to finish before it continues. " +
		"On create, Terraform waits for the first sync. " +
		"On update, Terraform first waits for a sync that is already in progress, then sends the update and waits for the sync that the update starts. " +
		"If the sync fails, the apply fails with the error from Infisical. " +
		"If the wait fails on create, Terraform marks the resource as tainted, and the next apply deletes and recreates the secret sync."

	if r.CanImportSecrets {
		description += " The recreated sync runs initial_sync_behavior again: the import behaviors bring secrets that exist only in the destination back into Infisical, " +
			"and import-prioritize-destination also overwrites Infisical values with the values in the destination."
	}

	description += " If the configuration is correct and only the destination needs a fix, run `terraform untaint` on the resource instead. " +
		"The wait has no effect when auto_sync_enabled is false, because Infisical then doesn't run a sync. " +
		fmt.Sprintf("Terraform waits up to the create and update values in timeouts (default: %s). ", defaultSecretSyncWaitTimeoutText) +
		"Defaults to false."

	return description
}

func (r *SecretSyncBaseResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	includeAllSubFoldersDescription := "Whether to sync secrets from folders beneath the source path as well."
	// Syncs without initial_sync_behavior always overwrite the destination, so the requirement only
	// applies to syncs that expose the setting.
	if _, ok := r.SyncOptionsAttributes["initial_sync_behavior"]; ok {
		includeAllSubFoldersDescription += " Requires initial_sync_behavior to be 'overwrite-destination'."
	}

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
			"include_all_sub_folders": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: includeAllSubFoldersDescription,
				Default:     booldefault.StaticBool(false),
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
			"wait_for_sync_completion": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: r.waitForSyncCompletionDescription(),
			},
			"timeouts": r.timeoutsAttribute(ctx),
		},
	}
}

func (r *SecretSyncBaseResource) timeoutsAttribute(ctx context.Context) schema.Attribute {
	attribute := timeouts.Attributes(ctx, timeouts.Opts{
		Create:            true,
		Update:            true,
		CreateDescription: fmt.Sprintf("How long to wait for the first sync when wait_for_sync_completion is true, as a positive duration string such as \"30s\" or \"15m\". Defaults to %s.", defaultSecretSyncWaitTimeoutText),
		UpdateDescription: fmt.Sprintf("How long to wait for syncs during an update when wait_for_sync_completion is true, as a positive duration string such as \"30s\" or \"15m\". Covers both the sync already in progress and the sync that the update starts. Defaults to %s.", defaultSecretSyncWaitTimeoutText),
	})

	nested, ok := attribute.(schema.SingleNestedAttribute)
	if !ok {
		return attribute
	}
	nested.Description = "How long Terraform waits for syncs when wait_for_sync_completion is true."
	return nested
}

// ValidateConfig runs the checks that need more than one attribute.
func (r *SecretSyncBaseResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	r.validateSyncWaitConfig(ctx, req, resp)
	r.validateIncludeAllSubFoldersConfig(ctx, req, resp)
}

// validateSyncWaitConfig rejects timeouts that end the wait before it starts, and warns about a wait that can't
// happen: with auto sync off, create and update don't queue a sync.
func (r *SecretSyncBaseResource) validateSyncWaitConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var configTimeouts timeouts.Value
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("timeouts"), &configTimeouts)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// The timeouts module only checks the syntax, and an unparsable value already has its own error.
	for _, name := range []string{"create", "update"} {
		get := configTimeouts.Create
		if name == "update" {
			get = configTimeouts.Update
		}
		if timeout, diags := get(ctx, defaultSecretSyncWaitTimeout); !diags.HasError() && timeout <= 0 {
			resp.Diagnostics.AddAttributeError(
				path.Root("timeouts").AtName(name),
				"Invalid timeout",
				fmt.Sprintf("timeouts.%s must be greater than zero, got %s.", name, timeout),
			)
		}
	}

	var waitForSyncCompletion, autoSyncEnabled types.Bool

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("wait_for_sync_completion"), &waitForSyncCompletion)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("auto_sync_enabled"), &autoSyncEnabled)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if waitForSyncCompletion.ValueBool() && !autoSyncEnabled.IsNull() && !autoSyncEnabled.IsUnknown() && !autoSyncEnabled.ValueBool() {
		resp.Diagnostics.AddAttributeWarning(
			path.Root("wait_for_sync_completion"),
			"wait_for_sync_completion has no effect",
			"auto_sync_enabled is false, so Infisical doesn't run a sync when the secret sync is created or updated, and Terraform has nothing to wait for.",
		)
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

func (r *SecretSyncBaseResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	r.planDeprecatedIncludeAllSubFolders(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil || !r.client.Config.IsMachineIdentityAuth {
		return
	}

	var plan SecretSyncBaseResourceModel
	if diags := req.Plan.Get(ctx, &plan); diags.HasError() {
		return
	}

	r.addOverwriteDestinationWarnings(ctx, plan, resp)

	if !req.State.Raw.IsNull() {
		var state SecretSyncBaseResourceModel
		if diags := req.State.Get(ctx, &state); !diags.HasError() {
			if plan.DestinationConfig.Equal(state.DestinationConfig) {
				return
			}
		}
	}

	var destinationConfigMap map[string]interface{}
	var configDiags diag.Diagnostics
	var excludeSyncId string

	if req.State.Raw.IsNull() {
		destinationConfigMap, configDiags = r.ReadDestinationConfigForCreateFromPlan(ctx, plan)
	} else {
		var state SecretSyncBaseResourceModel
		if diags := req.State.Get(ctx, &state); diags.HasError() {
			return
		}
		destinationConfigMap, configDiags = r.ReadDestinationConfigForUpdateFromPlan(ctx, plan, state)
		if !state.ID.IsNull() && !state.ID.IsUnknown() {
			excludeSyncId = state.ID.ValueString()
		}
	}

	if configDiags.HasError() {
		return
	}

	checkRequest := infisical.CheckDuplicateDestinationRequest{
		App:               r.App,
		DestinationConfig: destinationConfigMap,
		ProjectID:         plan.ProjectID.ValueString(),
		ExcludeSyncId:     excludeSyncId,
	}

	duplicateCheck, err := r.client.CheckDuplicateDestination(checkRequest)
	if err != nil {
		resp.Diagnostics.AddWarning(
			"Unable to check for duplicate destinations",
			"Warning: Could not verify if this destination already exists. This may result in conflicts. Error: "+err.Error(),
		)
	} else if duplicateCheck.HasDuplicate {
		resp.Diagnostics.AddWarning(
			"Duplicate destination configuration detected",
			"A secret sync with the same destination configuration already exists in this project. Proceeding with this operation may result in conflicts or unexpected behavior.",
		)
	}
}

// deprecatedIncludeAllSubFoldersPath points at the sync_options.include_all_sub_folders attribute some
// syncs (Spacelift) shipped before include_all_sub_folders moved to the root of the sync.
var deprecatedIncludeAllSubFoldersPath = path.Root("sync_options").AtName("include_all_sub_folders")

func (r *SecretSyncBaseResource) hasDeprecatedIncludeAllSubFolders() bool {
	_, ok := r.SyncOptionsAttributes["include_all_sub_folders"]
	return ok
}

// planDeprecatedIncludeAllSubFolders plans the root include_all_sub_folders from the deprecated
// sync_options.include_all_sub_folders when the root is left out of the config, so configs that still use
// the deprecated attribute keep including subfolders. The root always wins when it is set.
func (r *SecretSyncBaseResource) planDeprecatedIncludeAllSubFolders(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if !r.hasDeprecatedIncludeAllSubFolders() {
		return
	}

	var rootValue types.Bool
	var syncOptions types.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("include_all_sub_folders"), &rootValue)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("sync_options"), &syncOptions)...)
	if resp.Diagnostics.HasError() || !rootValue.IsNull() || syncOptions.IsNull() {
		return
	}

	// While the deprecated value is unknown the root stays unknown too. Planning it as false would let
	// the apply-time plan turn it into true, which Terraform rejects as an inconsistent final plan.
	includeAllSubFolders := types.BoolUnknown()
	if !syncOptions.IsUnknown() {
		var deprecatedValue types.Bool
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, deprecatedIncludeAllSubFoldersPath, &deprecatedValue)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if !deprecatedValue.IsUnknown() {
			includeAllSubFolders = types.BoolValue(deprecatedValue.ValueBool())
		}
	}
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("include_all_sub_folders"), includeAllSubFolders)...)
}

// keepDeprecatedIncludeAllSubFolders carries the deprecated sync_options.include_all_sub_folders over from
// the prior state. The API only reports the resolved value, which follows the root attribute whenever both
// are set, so reading it back would make a deprecated value that differs from the root drift forever. The
// root attribute is still read from the API, so real drift is caught there.
func keepDeprecatedIncludeAllSubFolders(ctx context.Context, fromApi, prior types.Object) (types.Object, diag.Diagnostics) {
	priorValue, ok := prior.Attributes()["include_all_sub_folders"]
	if fromApi.IsNull() || fromApi.IsUnknown() || !ok || priorValue.IsNull() || priorValue.IsUnknown() {
		return fromApi, nil
	}

	attrs := fromApi.Attributes()
	attrs["include_all_sub_folders"] = priorValue
	return types.ObjectValue(fromApi.AttributeTypes(ctx), attrs)
}

// validateIncludeAllSubFoldersConfig rejects including subfolders together with an import initial sync behavior,
// which the API refuses for every destination. Syncs without initial_sync_behavior always overwrite, so they can
// include subfolders freely.
func (r *SecretSyncBaseResource) validateIncludeAllSubFoldersConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	if _, ok := r.SyncOptionsAttributes["initial_sync_behavior"]; !ok {
		return
	}

	var includeAllSubFolders types.Bool
	var initialSyncBehavior types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("include_all_sub_folders"), &includeAllSubFolders)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("sync_options").AtName("initial_sync_behavior"), &initialSyncBehavior)...)
	if resp.Diagnostics.HasError() || !includeAllSubFolders.ValueBool() || initialSyncBehavior.IsNull() || initialSyncBehavior.IsUnknown() {
		return
	}

	if initialSyncBehavior.ValueString() != string(infisical.SecretSyncBehaviorOverwriteDestination) {
		resp.Diagnostics.AddAttributeError(
			path.Root("include_all_sub_folders"),
			"Invalid include_all_sub_folders",
			"include_all_sub_folders can only be enabled when initial_sync_behavior is 'overwrite-destination', because there is no single folder to import existing destination secrets into.",
		)
	}
}

// addOverwriteDestinationWarnings adds warnings when using overwrite destination initial sync behavior.
func (r *SecretSyncBaseResource) addOverwriteDestinationWarnings(ctx context.Context, plan SecretSyncBaseResourceModel, resp *resource.ModifyPlanResponse) {
	syncOptions, diags := r.ReadSyncOptionsForCreateFromPlan(ctx, plan)
	if diags.HasError() {
		return
	}

	initialSyncBehavior, _ := syncOptions["initialSyncBehavior"].(string)
	disableSecretDeletion, _ := syncOptions["disableSecretDeletion"].(bool)

	if disableSecretDeletion {
		return
	}

	if !r.CanImportSecrets {
		resp.Diagnostics.AddWarning(
			fmt.Sprintf("Existing secrets from %s may be removed", r.SyncName),
			fmt.Sprintf("Secrets not present in Infisical will be removed from the destination. Consider adding a key_schema or setting disable_secret_deletion to true if you do not want existing secrets to be removed from %s.", r.SyncName),
		)
	} else if initialSyncBehavior == string(infisical.SecretSyncBehaviorOverwriteDestination) {
		// Destination supports import but user chose overwrite - warn about secret deletion with import suggestion
		resp.Diagnostics.AddWarning(
			"Overwrite destination will remove secrets not present in Infisical",
			fmt.Sprintf("Secrets not present in Infisical will be removed from the destination. If you have secrets in %s that you do not want deleted, consider setting initial_sync_behavior to import-prioritize-source or import-prioritize-destination. Alternatively, configure a key_schema or set disable_secret_deletion to true to have Infisical ignore these secrets.", r.SyncName),
		)
	}
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

	secretSync, err := r.client.CreateSecretSync(infisical.CreateSecretSyncRequest{
		App:                  r.App,
		Name:                 plan.Name.ValueString(),
		Description:          plan.Description.ValueString(),
		ProjectID:            plan.ProjectID.ValueString(),
		ConnectionID:         plan.ConnectionID.ValueString(),
		Environment:          plan.Environment.ValueString(),
		SecretPath:           plan.SecretPath.ValueString(),
		IncludeAllSubFolders: plan.IncludeAllSubFolders.ValueBool(),
		AutoSyncEnabled:      plan.AutoSyncEnabled.ValueBool(),
		SyncOptions:          syncOptions,
		DestinationConfig:    destinationConfigMap,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating secret sync",
			"Couldn't create secret sync, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(secretSync.ID)

	// Save state before waiting, so a failed wait still tracks the secret sync that now exists.
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.WaitForSyncCompletion.ValueBool() || !secretSync.AutoSyncEnabled {
		return
	}

	timeout, diags := plan.Timeouts.Create(ctx, defaultSecretSyncWaitTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := r.waitForSyncResult(waitCtx, secretSync.ID); err != nil {
		resp.Diagnostics.AddError(
			"Error waiting for secret sync",
			fmt.Sprintf("The %s secret sync was created, but %s.\n\n"+
				"Terraform marks the resource as tainted, and the next apply deletes and recreates the secret sync. "+
				"If the configuration is correct and only the destination needs a fix, run `terraform untaint` on the resource, fix the destination, and trigger the sync again from Infisical.",
				r.SyncName, err.Error()),
		)
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

	if state.ID.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	// Import starts from an ID only. Set the default, so the next plan shows no change.
	if state.WaitForSyncCompletion.IsNull() {
		state.WaitForSyncCompletion = types.BoolValue(false)
	}

	secretSync, err := r.client.GetSecretSyncById(infisical.GetSecretSyncByIdRequest{
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
	state.IncludeAllSubFolders = types.BoolValue(secretSync.IncludeAllSubFolders)
	state.AutoSyncEnabled = types.BoolValue(secretSync.AutoSyncEnabled)

	if !(state.Description.IsNull() && secretSync.Description == "") {
		state.Description = types.StringValue(secretSync.Description)
	}

	priorSyncOptions := state.SyncOptions
	state.SyncOptions, diags = r.ReadSyncOptionsFromApi(ctx, secretSync)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.hasDeprecatedIncludeAllSubFolders() {
		state.SyncOptions, diags = keepDeprecatedIncludeAllSubFolders(ctx, state.SyncOptions, priorSyncOptions)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
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

	// Every PATCH queues a sync, so a change to a provider-only setting must not call the API.
	if !secretSyncApiFieldsChanged(plan, state) {
		diags = resp.State.Set(ctx, plan)
		resp.Diagnostics.Append(diags...)
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

	waitForSync := plan.WaitForSyncCompletion.ValueBool()
	waitCtx := ctx
	if waitForSync {
		timeout, diags := plan.Timeouts.Update(ctx, defaultSecretSyncWaitTimeout)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		// One deadline covers both waits, so the update never takes longer than the configured timeout.
		var cancel context.CancelFunc
		waitCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()

		if plan.AutoSyncEnabled.ValueBool() {
			if err := r.waitForSyncIdle(waitCtx, state.ID.ValueString()); err != nil {
				resp.Diagnostics.AddError(
					"Error updating secret sync",
					fmt.Sprintf("Couldn't update the %s secret sync: %s. Terraform did not send the update.", r.SyncName, err.Error()),
				)
				return
			}
			// The final read at the deadline can report the earlier sync as finished. The timeout is used up by then,
			// so an update sent now could only end in a timeout error after it was applied.
			if waitCtx.Err() != nil {
				resp.Diagnostics.AddError(
					"Error updating secret sync",
					fmt.Sprintf("Couldn't update the %s secret sync: the update timeout ran out while waiting for the sync that was already in progress to finish. Terraform did not send the update. To wait longer, increase the values in timeouts.", r.SyncName),
				)
				return
			}
		}
	}

	updatedSecretSync, err := r.client.UpdateSecretSync(infisical.UpdateSecretSyncRequest{
		App:                  r.App,
		ID:                   state.ID.ValueString(),
		Name:                 plan.Name.ValueString(),
		Description:          plan.Description.ValueString(),
		ProjectID:            plan.ProjectID.ValueString(),
		ConnectionID:         plan.ConnectionID.ValueString(),
		Environment:          plan.Environment.ValueString(),
		SecretPath:           plan.SecretPath.ValueString(),
		IncludeAllSubFolders: plan.IncludeAllSubFolders.ValueBool(),
		AutoSyncEnabled:      plan.AutoSyncEnabled.ValueBool(),
		SyncOptions:          secretSyncPlan,
		DestinationConfig:    destinationConfigMap,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating secret sync",
			"Couldn't update secret sync, unexpected error: "+err.Error(),
		)
		return
	}

	// Save state before waiting: the update is applied, whatever the sync result is.
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !waitForSync || !updatedSecretSync.AutoSyncEnabled {
		return
	}

	if err := r.waitForSyncResult(waitCtx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error waiting for secret sync",
			fmt.Sprintf("The %s secret sync was updated, but %s.", r.SyncName, err.Error()),
		)
	}
}

// secretSyncApiFieldsChanged reports whether the plan changes anything that the API stores.
func secretSyncApiFieldsChanged(plan, state SecretSyncBaseResourceModel) bool {
	return !plan.Name.Equal(state.Name) ||
		!plan.Description.Equal(state.Description) ||
		!plan.ProjectID.Equal(state.ProjectID) ||
		!plan.ConnectionID.Equal(state.ConnectionID) ||
		!plan.Environment.Equal(state.Environment) ||
		!plan.SecretPath.Equal(state.SecretPath) ||
		!plan.IncludeAllSubFolders.Equal(state.IncludeAllSubFolders) ||
		!plan.AutoSyncEnabled.Equal(state.AutoSyncEnabled) ||
		!plan.SyncOptions.Equal(state.SyncOptions) ||
		!plan.DestinationConfig.Equal(state.DestinationConfig)
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
