package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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
	client                                 *infisical.Client
	DestinationConfigAttributes            map[string]schema.Attribute
	ReadDestinationConfigForCreateFromPlan func(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics)
	ReadDestinationConfigForUpdateFromPlan func(ctx context.Context, plan SecretSyncBaseResourceModel, state SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics)
	ReadDestinationConfigFromApi           func(ctx context.Context, secretSync infisical.SecretSync) (types.Object, diag.Diagnostics)

	SyncOptionsAttributes            map[string]schema.Attribute
	ReadSyncOptionsForCreateFromPlan func(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics)
	ReadSyncOptionsForUpdateFromPlan func(ctx context.Context, plan SecretSyncBaseResourceModel, state SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics)
	ReadSyncOptionsFromApi           func(ctx context.Context, secretSync infisical.SecretSync) (types.Object, diag.Diagnostics)
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

			"sync_options": schema.StringAttribute{
				Required: true,
				Description: buildJSONSchemaDescription(
					"Parameters to modify how secrets are synced.",
					r.SyncOptionsAttributes,
				),
			},

			"destination_config": schema.StringAttribute{
				Required: true,
				Description: buildJSONSchemaDescription(
					"The destination configuration for the secret sync.",
					r.DestinationConfigAttributes,
				),
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

	var syncOptions map[string]interface{}
	var destinationConfigMap map[string]interface{}

	// first we parse sync_options and destination_config from the plan
	var syncOptsJSON types.String
	diags := req.Plan.GetAttribute(ctx, path.Root("sync_options"), &syncOptsJSON)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := json.Unmarshal([]byte(syncOptsJSON.ValueString()), &syncOptions); err != nil {
		resp.Diagnostics.AddError("Invalid JSON", fmt.Sprintf("Failed to parse sync_options: %s", err.Error()))
		return
	}
	// transform sync_options since we're not calling a function for it
	syncOptions = transformMapKeys(syncOptions)

	var destConfigJSON types.String
	diags = req.Plan.GetAttribute(ctx, path.Root("destination_config"), &destConfigJSON)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var tempDestConfigMap map[string]interface{}
	if err := json.Unmarshal([]byte(destConfigJSON.ValueString()), &tempDestConfigMap); err != nil {
		resp.Diagnostics.AddError("Invalid JSON", fmt.Sprintf("Failed to parse destination_config: %s", err.Error()))
		return
	}

	// convert map to types.Object (with snake_case keys because this is the convention we follow for all map attributes across all secret syncs)
	destConfigObj, d := mapToTypesObject(tempDestConfigMap, r.DestinationConfigAttributes)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	// create temporary plan model that we pass into the ReadDestinationConfigForCreateFromPlan function for the validation
	tempPlan := SecretSyncBaseResourceModel{
		DestinationConfig: destConfigObj,
		SyncOptions:       types.ObjectNull(nil),
	}

	// This function validates with snake_case keys and transforms to API format (camelCase)
	destinationConfigMap, diags = r.ReadDestinationConfigForCreateFromPlan(ctx, tempPlan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// get the rest of the plan fields (common to both modes)
	// we have to get them individually because if we try to read the whole plan at once, we'll get type errors when trying to read the sync_options and destination_config attributes. there's no pretty way to do this
	var planName types.String
	var planDescription types.String
	var planProjectID types.String
	var planConnectionID types.String
	var planEnvironment types.String
	var planSecretPath types.String
	var planAutoSyncEnabled types.Bool

	req.Plan.GetAttribute(ctx, path.Root("name"), &planName)
	req.Plan.GetAttribute(ctx, path.Root("description"), &planDescription)
	req.Plan.GetAttribute(ctx, path.Root("project_id"), &planProjectID)
	req.Plan.GetAttribute(ctx, path.Root("connection_id"), &planConnectionID)
	req.Plan.GetAttribute(ctx, path.Root("environment"), &planEnvironment)
	req.Plan.GetAttribute(ctx, path.Root("secret_path"), &planSecretPath)
	req.Plan.GetAttribute(ctx, path.Root("auto_sync_enabled"), &planAutoSyncEnabled)

	// Validation
	initialSyncBehavior, ok := syncOptions["initialSyncBehavior"].(string)
	if !ok || initialSyncBehavior == "" {
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

	secretSync, err := r.client.CreateSecretSync(infisical.CreateSecretSyncRequest{
		App:               r.App,
		Name:              planName.ValueString(),
		Description:       planDescription.ValueString(),
		ProjectID:         planProjectID.ValueString(),
		ConnectionID:      planConnectionID.ValueString(),
		Environment:       planEnvironment.ValueString(),
		SecretPath:        planSecretPath.ValueString(),
		AutoSyncEnabled:   planAutoSyncEnabled.ValueBool(),
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

	// set the state
	var plan SecretSyncBaseResourceModel
	plan.ID = types.StringValue(secretSync.ID)
	plan.Name = planName
	plan.Description = planDescription
	plan.ProjectID = planProjectID
	plan.ConnectionID = planConnectionID
	plan.Environment = planEnvironment
	plan.SecretPath = planSecretPath
	plan.AutoSyncEnabled = planAutoSyncEnabled

	// get the original json strings from plan
	var syncOptsJSONOriginal types.String
	req.Plan.GetAttribute(ctx, path.Root("sync_options"), &syncOptsJSONOriginal)
	resp.State.SetAttribute(ctx, path.Root("sync_options"), syncOptsJSONOriginal.ValueString())

	var destConfigJSONOriginal types.String
	req.Plan.GetAttribute(ctx, path.Root("destination_config"), &destConfigJSONOriginal)
	resp.State.SetAttribute(ctx, path.Root("destination_config"), destConfigJSONOriginal.ValueString())

	plan.SyncOptions = types.ObjectNull(nil)
	plan.DestinationConfig = types.ObjectNull(nil)

	// set other fields (again no pretty way to do this)
	resp.State.SetAttribute(ctx, path.Root("id"), plan.ID)
	resp.State.SetAttribute(ctx, path.Root("name"), plan.Name)
	resp.State.SetAttribute(ctx, path.Root("description"), plan.Description)
	resp.State.SetAttribute(ctx, path.Root("project_id"), plan.ProjectID)
	resp.State.SetAttribute(ctx, path.Root("connection_id"), plan.ConnectionID)
	resp.State.SetAttribute(ctx, path.Root("environment"), plan.Environment)
	resp.State.SetAttribute(ctx, path.Root("secret_path"), plan.SecretPath)
	resp.State.SetAttribute(ctx, path.Root("auto_sync_enabled"), plan.AutoSyncEnabled)

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

	var stateID types.String
	diags := req.State.GetAttribute(ctx, path.Root("id"), &stateID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	secretSync, err := r.client.GetSecretSyncById(infisical.GetSecretSyncByIdRequest{
		App: r.App,
		ID:  stateID.ValueString(),
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

	// get current state values to preserve structure
	var currentSyncOpts types.String
	var currentDestConfig types.String
	req.State.GetAttribute(ctx, path.Root("sync_options"), &currentSyncOpts)
	req.State.GetAttribute(ctx, path.Root("destination_config"), &currentDestConfig)

	var currentSyncOptsMap map[string]interface{}
	if err := json.Unmarshal([]byte(currentSyncOpts.ValueString()), &currentSyncOptsMap); err != nil {
		resp.Diagnostics.AddError("Invalid JSON", fmt.Sprintf("Failed to parse sync_options: %s", err.Error()))
		return
	}

	var currentDestConfigMap map[string]interface{}
	if err := json.Unmarshal([]byte(currentDestConfig.ValueString()), &currentDestConfigMap); err != nil {
		resp.Diagnostics.AddError("Invalid JSON", fmt.Sprintf("Failed to parse destination_config: %s", err.Error()))
		return
	}

	syncOptsObj, d := r.ReadSyncOptionsFromApi(ctx, secretSync)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	destConfigObj, d := r.ReadDestinationConfigFromApi(ctx, secretSync)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	// update only the keys that exist in current state
	for k := range currentSyncOptsMap {
		if attr, ok := syncOptsObj.Attributes()[k]; ok {
			currentSyncOptsMap[k] = extractAttrValue(attr)
		}
	}

	for k := range currentDestConfigMap {
		if attr, ok := destConfigObj.Attributes()[k]; ok {
			currentDestConfigMap[k] = extractAttrValue(attr)
		}
	}

	syncOptsJSON, err := json.Marshal(currentSyncOptsMap)
	if err != nil {
		resp.Diagnostics.AddError("Error marshalling sync options", "Couldn't marshal sync options, unexpected error: "+err.Error())
		return
	}
	destConfigJSON, err := json.Marshal(currentDestConfigMap)
	if err != nil {
		resp.Diagnostics.AddError("Error marshalling destination config", "Couldn't marshal destination config, unexpected error: "+err.Error())
		return
	}

	finalSyncOpts := string(syncOptsJSON)
	if areJSONEquivalent(currentSyncOpts.ValueString(), finalSyncOpts) {
		finalSyncOpts = currentSyncOpts.ValueString() // Keep original formatting
	}

	finalDestConfig := string(destConfigJSON)
	if areJSONEquivalent(currentDestConfig.ValueString(), finalDestConfig) {
		finalDestConfig = currentDestConfig.ValueString() // Keep original formatting
	}

	// Set all attributes
	resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(secretSync.ID))
	resp.State.SetAttribute(ctx, path.Root("connection_id"), types.StringValue(secretSync.Connection.ConnectionID))
	resp.State.SetAttribute(ctx, path.Root("name"), types.StringValue(secretSync.Name))
	resp.State.SetAttribute(ctx, path.Root("project_id"), types.StringValue(secretSync.ProjectID))
	resp.State.SetAttribute(ctx, path.Root("environment"), types.StringValue(secretSync.Environment.Slug))
	resp.State.SetAttribute(ctx, path.Root("secret_path"), types.StringValue(secretSync.SecretFolder.Path))
	resp.State.SetAttribute(ctx, path.Root("auto_sync_enabled"), types.BoolValue(secretSync.AutoSyncEnabled))
	resp.State.SetAttribute(ctx, path.Root("description"), types.StringValue(secretSync.Description))
	resp.State.SetAttribute(ctx, path.Root("sync_options"), finalSyncOpts)
	resp.State.SetAttribute(ctx, path.Root("destination_config"), finalDestConfig)

}

func (r *SecretSyncBaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to update secret sync",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var secretSyncPlan map[string]interface{}
	var destinationConfigMap map[string]interface{}
	var stateID types.String

	diags := req.State.GetAttribute(ctx, path.Root("id"), &stateID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// parse sync_options and transform keys (snake to camel for API calls)
	var syncOptsJSON types.String
	diags = req.Plan.GetAttribute(ctx, path.Root("sync_options"), &syncOptsJSON)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := json.Unmarshal([]byte(syncOptsJSON.ValueString()), &secretSyncPlan); err != nil {
		resp.Diagnostics.AddError("Invalid JSON", fmt.Sprintf("Failed to parse sync_options: %s", err.Error()))
		return
	}
	secretSyncPlan = transformMapKeys(secretSyncPlan)

	var destConfigJSON types.String
	diags = req.Plan.GetAttribute(ctx, path.Root("destination_config"), &destConfigJSON)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var tempDestConfigMap map[string]interface{}
	if err := json.Unmarshal([]byte(destConfigJSON.ValueString()), &tempDestConfigMap); err != nil {
		resp.Diagnostics.AddError("Invalid JSON", fmt.Sprintf("Failed to parse destination_config: %s", err.Error()))
		return
	}
	// we gotta keep the snake_case keys for any potential validation like in the github secret sync
	destConfigObj, d := mapToTypesObject(tempDestConfigMap, r.DestinationConfigAttributes)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	var stateDestConfigJSON types.String
	diags = req.State.GetAttribute(ctx, path.Root("destination_config"), &stateDestConfigJSON)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var tempStateDestConfigMap map[string]interface{}
	if err := json.Unmarshal([]byte(stateDestConfigJSON.ValueString()), &tempStateDestConfigMap); err != nil {
		resp.Diagnostics.AddError("Invalid JSON", fmt.Sprintf("Failed to parse state destination_config: %s", err.Error()))
		return
	}

	stateDestConfigObj, d := mapToTypesObject(tempStateDestConfigMap, r.DestinationConfigAttributes)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	tempPlan := SecretSyncBaseResourceModel{
		DestinationConfig: destConfigObj,
		SyncOptions:       types.ObjectNull(nil),
	}

	tempState := SecretSyncBaseResourceModel{
		DestinationConfig: stateDestConfigObj,
		SyncOptions:       types.ObjectNull(nil),
	}

	destinationConfigMap, diags = r.ReadDestinationConfigForUpdateFromPlan(ctx, tempPlan, tempState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validation
	initialSyncBehavior, ok := secretSyncPlan["initialSyncBehavior"].(string)
	if !ok || initialSyncBehavior == "" {
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

	// Get other plan fields
	var planName, planDescription, planProjectID, planConnectionID, planEnvironment, planSecretPath types.String
	var planAutoSyncEnabled types.Bool

	req.Plan.GetAttribute(ctx, path.Root("name"), &planName)
	req.Plan.GetAttribute(ctx, path.Root("description"), &planDescription)
	req.Plan.GetAttribute(ctx, path.Root("project_id"), &planProjectID)
	req.Plan.GetAttribute(ctx, path.Root("connection_id"), &planConnectionID)
	req.Plan.GetAttribute(ctx, path.Root("environment"), &planEnvironment)
	req.Plan.GetAttribute(ctx, path.Root("secret_path"), &planSecretPath)
	req.Plan.GetAttribute(ctx, path.Root("auto_sync_enabled"), &planAutoSyncEnabled)

	_, err := r.client.UpdateSecretSync(infisical.UpdateSecretSyncRequest{
		App:               r.App,
		ID:                stateID.ValueString(),
		Name:              planName.ValueString(),
		Description:       planDescription.ValueString(),
		ProjectID:         planProjectID.ValueString(),
		ConnectionID:      planConnectionID.ValueString(),
		Environment:       planEnvironment.ValueString(),
		SecretPath:        planSecretPath.ValueString(),
		AutoSyncEnabled:   planAutoSyncEnabled.ValueBool(),
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

	// get original JSON strings from plan to preserve formatting
	var syncOptsJSONOriginal types.String
	var destConfigJSONOriginal types.String
	req.Plan.GetAttribute(ctx, path.Root("sync_options"), &syncOptsJSONOriginal)
	req.Plan.GetAttribute(ctx, path.Root("destination_config"), &destConfigJSONOriginal)

	resp.State.SetAttribute(ctx, path.Root("id"), stateID)
	resp.State.SetAttribute(ctx, path.Root("name"), planName)
	resp.State.SetAttribute(ctx, path.Root("description"), planDescription)
	resp.State.SetAttribute(ctx, path.Root("project_id"), planProjectID)
	resp.State.SetAttribute(ctx, path.Root("connection_id"), planConnectionID)
	resp.State.SetAttribute(ctx, path.Root("environment"), planEnvironment)
	resp.State.SetAttribute(ctx, path.Root("secret_path"), planSecretPath)
	resp.State.SetAttribute(ctx, path.Root("auto_sync_enabled"), planAutoSyncEnabled)
	resp.State.SetAttribute(ctx, path.Root("sync_options"), syncOptsJSON.ValueString())
	resp.State.SetAttribute(ctx, path.Root("destination_config"), destConfigJSON.ValueString())
}

func (r *SecretSyncBaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !r.client.Config.IsMachineIdentityAuth {
		resp.Diagnostics.AddError(
			"Unable to delete secret sync",
			"Only Machine Identity authentication is supported for this operation",
		)
		return
	}

	var stateID types.String
	diags := req.State.GetAttribute(ctx, path.Root("id"), &stateID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteSecretSync(infisical.DeleteSecretSyncRequest{
		App: r.App,
		ID:  stateID.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting secret sync",
			"Couldn't delete secret sync from Infisical, unexpected error: "+err.Error(),
		)
	}
}

func snakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

func transformMapKeys(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		camelKey := snakeToCamel(k)

		// incase of nested maps we recursively transform the keys
		if nestedMap, ok := v.(map[string]interface{}); ok {
			result[camelKey] = transformMapKeys(nestedMap)
		} else {
			result[camelKey] = v
		}
	}
	return result
}

func extractAttrValue(v attr.Value) interface{} {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}

	switch val := v.(type) {
	case types.String:
		return val.ValueString()
	case types.Bool:
		return val.ValueBool()
	case types.Int64:
		return val.ValueInt64()
	case types.Float64:
		return val.ValueFloat64()
	case types.List:
		result := make([]interface{}, 0)
		for _, elem := range val.Elements() {
			result = append(result, extractAttrValue(elem))
		}
		return result
	case types.Object:
		result := make(map[string]interface{})
		for k, elem := range val.Attributes() {
			result[k] = extractAttrValue(elem)
		}
		return result
	default:
		return nil
	}
}

func areJSONEquivalent(json1, json2 string) bool {
	var obj1, obj2 interface{}

	if err := json.Unmarshal([]byte(json1), &obj1); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(json2), &obj2); err != nil {
		return false
	}

	return reflect.DeepEqual(obj1, obj2)
}

// helper for converting map[string] to types.Object.
func mapToTypesObject(m map[string]interface{}, attrTypes map[string]schema.Attribute) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	attrTypeMap := make(map[string]attr.Type)
	attrValues := make(map[string]attr.Value)

	for key, schemaAttr := range attrTypes {
		// Determine the type from schema
		switch a := schemaAttr.(type) {
		case schema.StringAttribute:
			attrTypeMap[key] = types.StringType
			if val, ok := m[key].(string); ok {
				attrValues[key] = types.StringValue(val)
			} else {
				attrValues[key] = types.StringNull()
			}
		case schema.BoolAttribute:
			attrTypeMap[key] = types.BoolType
			if val, ok := m[key].(bool); ok {
				attrValues[key] = types.BoolValue(val)
			} else {
				attrValues[key] = types.BoolNull()
			}
		case schema.Int64Attribute:
			attrTypeMap[key] = types.Int64Type
			if val, ok := m[key].(float64); ok {
				attrValues[key] = types.Int64Value(int64(val))
			} else {
				attrValues[key] = types.Int64Null()
			}
		case schema.ListAttribute:
			attrTypeMap[key] = types.ListType{ElemType: a.ElementType}
			if val, ok := m[key].([]interface{}); ok {
				elemValues := make([]attr.Value, len(val))
				for i, v := range val {
					if a.ElementType == types.Int64Type {
						if fVal, ok := v.(float64); ok {
							elemValues[i] = types.Int64Value(int64(fVal))
						}
					}
				}
				listVal, d := types.ListValue(a.ElementType, elemValues)
				diags.Append(d...)
				attrValues[key] = listVal
			} else {
				attrValues[key] = types.ListNull(a.ElementType)
			}
		}
	}

	obj, d := types.ObjectValue(attrTypeMap, attrValues)
	diags.Append(d...)
	return obj, diags
}

func buildJSONSchemaDescription(baseDesc string, attrs map[string]schema.Attribute) string {
	var fields []string

	for name, attr := range attrs {
		var fieldDesc string
		var required string

		switch a := attr.(type) {
		case schema.StringAttribute:
			if a.Required {
				required = "required"
			} else {
				required = "optional"
			}
			fieldDesc = fmt.Sprintf("  - `%s` (%s): %s", name, required, a.Description)

		case schema.BoolAttribute:
			if a.Required {
				required = "required"
			} else {
				required = "optional"
			}
			fieldDesc = fmt.Sprintf("  - `%s` (%s): %s", name, required, a.Description)

		case schema.Int64Attribute:
			if a.Required {
				required = "required"
			} else {
				required = "optional"
			}
			fieldDesc = fmt.Sprintf("  - `%s` (%s): %s", name, required, a.Description)

		case schema.ListAttribute:
			if a.Required {
				required = "required"
			} else {
				required = "optional"
			}
			fieldDesc = fmt.Sprintf("  - `%s` (%s): %s", name, required, a.Description)
		}

		if fieldDesc != "" {
			fields = append(fields, fieldDesc)
		}
	}

	// Sort alphabetically for consistency
	sort.Strings(fields)

	return fmt.Sprintf("%s Must be a JSON string with the following structure:\n\n%s",
		baseDesc, strings.Join(fields, "\n"))
}
