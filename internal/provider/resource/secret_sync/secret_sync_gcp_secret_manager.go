package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const (
	gcpSecretManagerScopeGlobal = "global"
	gcpSecretManagerScopeRegion = "region"
)

// SecretSyncGcpResourceModel describes the data source data model.
type SecretSyncGcpSecretManagerDestinationConfigModel struct {
	ProjectID  types.String `tfsdk:"project_id"`
	Scope      types.String `tfsdk:"scope"`
	LocationID types.String `tfsdk:"location_id"`
}

type SecretSyncGcpSecretManagerSyncOptionsModel struct {
	InitialSyncBehavior   types.String `tfsdk:"initial_sync_behavior"`
	DisableSecretDeletion types.Bool   `tfsdk:"disable_secret_deletion"`
	KeySchema             types.String `tfsdk:"key_schema"`
}

// populateGcpSecretManagerDestinationConfig validates the scope/location_id
// combination and writes the resulting fields into destinationConfig.
func populateGcpSecretManagerDestinationConfig(destinationConfig map[string]interface{}, gcpConfig SecretSyncGcpSecretManagerDestinationConfigModel) error {
	scope := gcpConfig.Scope.ValueString()
	locationId := gcpConfig.LocationID.ValueString()
	hasLocationId := !gcpConfig.LocationID.IsNull() && locationId != ""

	switch scope {
	case gcpSecretManagerScopeGlobal:
		if hasLocationId {
			return fmt.Errorf("location_id must not be set when scope is 'global'")
		}
	case gcpSecretManagerScopeRegion:
		if !hasLocationId {
			return fmt.Errorf("location_id is required when scope is 'region'")
		}
		destinationConfig["locationId"] = locationId
	default:
		return fmt.Errorf("invalid value for scope field. Possible values are: global, region")
	}

	destinationConfig["scope"] = scope
	destinationConfig["projectId"] = gcpConfig.ProjectID.ValueString()

	return nil
}

func NewSecretSyncGcpSecretManagerResource() resource.Resource {
	return &SecretSyncBaseResource{
		App:              infisical.SecretSyncAppGCPSecretManager,
		SyncName:         "GCP Secret Manager",
		ResourceTypeName: "_secret_sync_gcp_secret_manager",
		AppConnection:    "GCP",
		CanImportSecrets: true,
		SyncOptionsAttributes: map[string]schema.Attribute{
			"initial_sync_behavior": schema.StringAttribute{
				Required:    true,
				Description: "Specify how Infisical should resolve the initial sync to the destination. Supported options: overwrite-destination, import-prioritize-source, import-prioritize-destination",
			},
			"disable_secret_deletion": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "When set to true, Infisical will not remove secrets from GCP Secret Manager. Enable this option if you intend to manage some secrets manually outside of Infisical.",
				Default:     booldefault.StaticBool(false),
			},
			"key_schema": schema.StringAttribute{
				Optional:    true,
				Description: "The format to use for structuring secret keys in the GCP Secret Manager destination.",
			},
		},
		DestinationConfigAttributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the GCP project to sync with",
			},
			"scope": schema.StringAttribute{
				Optional:    true,
				Description: "The scope of the sync with GCP Secret Manager. Supported options: global, region",
				Default:     stringdefault.StaticString("global"),
				Computed:    true,
			},
			"location_id": schema.StringAttribute{
				Optional:    true,
				Description: "The GCP region to sync secrets to (e.g. us-east1). Required when scope is 'region' and must not be set when scope is 'global'.",
			},
		},

		ReadSyncOptionsForCreateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			syncOptionsMap := make(map[string]interface{})

			var syncOptions SecretSyncGcpSecretManagerSyncOptionsModel
			diags := plan.SyncOptions.As(ctx, &syncOptions, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			syncOptionsMap["initialSyncBehavior"] = syncOptions.InitialSyncBehavior.ValueString()
			syncOptionsMap["disableSecretDeletion"] = syncOptions.DisableSecretDeletion.ValueBool()
			syncOptionsMap["keySchema"] = syncOptions.KeySchema.ValueString()
			return syncOptionsMap, nil

		},

		ReadSyncOptionsForUpdateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel, state SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			syncOptionsMap := make(map[string]interface{})

			var syncOptions SecretSyncGcpSecretManagerSyncOptionsModel
			diags := plan.SyncOptions.As(ctx, &syncOptions, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			syncOptionsMap["initialSyncBehavior"] = syncOptions.InitialSyncBehavior.ValueString()
			syncOptionsMap["disableSecretDeletion"] = syncOptions.DisableSecretDeletion.ValueBool()
			syncOptionsMap["keySchema"] = syncOptions.KeySchema.ValueString()
			return syncOptionsMap, nil
		},

		ReadSyncOptionsFromApi: func(ctx context.Context, secretSync infisical.SecretSync) (types.Object, diag.Diagnostics) {

			initialSyncBehavior, ok := secretSync.SyncOptions["initialSyncBehavior"].(string)
			if !ok {
				initialSyncBehavior = ""
			}

			disableSecretDeletion, ok := secretSync.SyncOptions["disableSecretDeletion"].(bool)
			if !ok {
				disableSecretDeletion = false
			}

			syncOptionsMap := map[string]attr.Value{
				"initial_sync_behavior":   types.StringValue(initialSyncBehavior),
				"disable_secret_deletion": types.BoolValue(disableSecretDeletion),
			}

			keySchema, ok := secretSync.SyncOptions["keySchema"].(string)
			if keySchema == "" || !ok {
				syncOptionsMap["key_schema"] = types.StringNull()
			} else {
				syncOptionsMap["key_schema"] = types.StringValue(keySchema)
			}

			return types.ObjectValue(map[string]attr.Type{
				"initial_sync_behavior":   types.StringType,
				"disable_secret_deletion": types.BoolType,
				"key_schema":              types.StringType,
			}, syncOptionsMap)
		},

		ReadDestinationConfigForCreateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			destinationConfig := make(map[string]interface{})

			var gcpConfig SecretSyncGcpSecretManagerDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &gcpConfig, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if err := populateGcpSecretManagerDestinationConfig(destinationConfig, gcpConfig); err != nil {
				diags.AddError("Unable to create GCP secret manager secret sync", err.Error())
				return nil, diags
			}

			return destinationConfig, diags
		},
		ReadDestinationConfigForUpdateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel, _ SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			destinationConfig := make(map[string]interface{})

			var gcpConfig SecretSyncGcpSecretManagerDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &gcpConfig, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if err := populateGcpSecretManagerDestinationConfig(destinationConfig, gcpConfig); err != nil {
				diags.AddError("Unable to update GCP secret manager secret sync", err.Error())
				return nil, diags
			}

			return destinationConfig, diags
		},
		ReadDestinationConfigFromApi: func(ctx context.Context, secretSync infisical.SecretSync) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			scopeVal, ok := secretSync.DestinationConfig["scope"].(string)
			if !ok {
				diags.AddError(
					"Invalid scope type",
					"Expected 'scope' to be a string but got something else",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			projectIdVal, ok := secretSync.DestinationConfig["projectId"].(string)
			if !ok {
				diags.AddError(
					"Invalid projectId type",
					"Expected 'projectId' to be a string but got something else",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			destinationConfig := map[string]attr.Value{
				"scope":      types.StringValue(scopeVal),
				"project_id": types.StringValue(projectIdVal),
			}

			locationIdVal, ok := secretSync.DestinationConfig["locationId"].(string)
			if !ok || locationIdVal == "" {
				destinationConfig["location_id"] = types.StringNull()
			} else {
				destinationConfig["location_id"] = types.StringValue(locationIdVal)
			}

			return types.ObjectValue(map[string]attr.Type{
				"scope":       types.StringType,
				"project_id":  types.StringType,
				"location_id": types.StringType,
			}, destinationConfig)
		},
	}
}
