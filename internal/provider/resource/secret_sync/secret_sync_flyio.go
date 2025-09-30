package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"
	"terraform-provider-infisical/internal/pkg/terraform"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func verifyFlyioDestinationConfigState(destinationConfig map[string]attr.Value, diags *diag.Diagnostics) bool {
	appIdAttr, exists := destinationConfig["app_id"]
	if !exists {
		diags.AddError("Invalid destination config", "Expected 'app_id' to be present")
		return false
	}

	appIdVal, ok := appIdAttr.(types.String)
	if !ok {
		diags.AddError("Invalid destination config", "Expected 'app_id' to be a string type")
		return false
	}

	if appIdVal.IsNull() || appIdVal.IsUnknown() {
		diags.AddError("Invalid destination config", "Expected 'app_id' to have a value")
		return false
	}

	requiredFields := []string{"app_id"}

	// Check required fields are not empty
	for _, field := range requiredFields {
		value, exists := destinationConfig[field]
		if !exists {
			diags.AddError("Invalid destination config", fmt.Sprintf("Expected '%s' to be present", field))
			return false
		}

		if terraform.IsAttrValueEmpty(value) {
			diags.AddError("Invalid destination config", fmt.Sprintf("Expected '%s' to be set", field))
			return false
		}
	}

	// Check for unexpected fields
	allowedFieldsMap := make(map[string]bool)
	for _, field := range requiredFields {
		allowedFieldsMap[field] = true
	}

	for field := range destinationConfig {
		if !allowedFieldsMap[field] {
			if terraform.IsAttrValueEmpty(destinationConfig[field]) {
				continue
			}

			diags.AddError("Invalid destination config", fmt.Sprintf("Unexpected field '%s'. Supported destination_config fields are: %v", field, requiredFields))
			return false
		}
	}

	return true
}

type SecretSyncFlyioDestinationConfigModel struct {
	AppId types.String `tfsdk:"app_id"`
}

type SecretSyncFlyioSyncOptionsModel struct {
	InitialSyncBehavior   types.String `tfsdk:"initial_sync_behavior"`
	DisableSecretDeletion types.Bool   `tfsdk:"disable_secret_deletion"`
	KeySchema             types.String `tfsdk:"key_schema"`
}

func NewSecretSyncFlyioResource() resource.Resource {
	return &SecretSyncBaseResource{
		CrossplaneCompatible: false,
		App:                  infisical.SecretSyncAppFlyio,
		SyncName:             "Fly.io",
		ResourceTypeName:     "_secret_sync_flyio",
		AppConnection:        infisical.AppConnectionAppFlyio,
		DestinationConfigAttributes: map[string]schema.Attribute{
			"app_id": schema.StringAttribute{
				Required:    true,
				Description: "The Fly.io app ID to sync secrets to.",
			},
		},
		SyncOptionsAttributes: map[string]schema.Attribute{
			"initial_sync_behavior": schema.StringAttribute{
				Required:    true,
				Description: "Specify how Infisical should resolve the initial sync to the destination. Supported options: overwrite-destination",
			},
			"disable_secret_deletion": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "When set to true, Infisical will not remove secrets from Fly.io. Enable this option if you intend to manage some secrets manually outside of Infisical.",
				Default:     booldefault.StaticBool(false),
			},
			"key_schema": schema.StringAttribute{
				Optional:    true,
				Description: "The format to use for structuring secret keys in the Fly.io destination.",
			},
		},

		ReadSyncOptionsForCreateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			syncOptionsMap := make(map[string]interface{})

			var syncOptions SecretSyncFlyioSyncOptionsModel
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
			syncOptionsMap := make(map[string]attr.Value)

			initialSyncBehavior, ok := secretSync.SyncOptions["initialSyncBehavior"].(string)
			if !ok {
				initialSyncBehavior = ""
			}

			disableSecretDeletion, ok := secretSync.SyncOptions["disableSecretDeletion"].(bool)
			if !ok {
				disableSecretDeletion = false
			}

			syncOptionsMap["initial_sync_behavior"] = types.StringValue(initialSyncBehavior)
			syncOptionsMap["disable_secret_deletion"] = types.BoolValue(disableSecretDeletion)

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

		ReadSyncOptionsForUpdateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel, state SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			syncOptionsMap := make(map[string]interface{})

			var syncOptions SecretSyncFlyioSyncOptionsModel
			diags := plan.SyncOptions.As(ctx, &syncOptions, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			syncOptionsMap["initialSyncBehavior"] = syncOptions.InitialSyncBehavior.ValueString()
			syncOptionsMap["disableSecretDeletion"] = syncOptions.DisableSecretDeletion.ValueBool()
			syncOptionsMap["keySchema"] = syncOptions.KeySchema.ValueString()

			return syncOptionsMap, nil
		},

		ReadDestinationConfigForCreateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			destinationConfig := make(map[string]interface{})

			var cfg SecretSyncFlyioDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &cfg, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			rawFlyioCfg := map[string]attr.Value{
				"app_id": cfg.AppId,
			}

			if !verifyFlyioDestinationConfigState(rawFlyioCfg, &diags) {
				return nil, diags
			}

			destinationConfig["appId"] = cfg.AppId.ValueString()

			return destinationConfig, diags
		},
		ReadDestinationConfigForUpdateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel, _ SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			destinationConfig := make(map[string]interface{})

			var cfg SecretSyncFlyioDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &cfg, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			rawFlyioCfg := map[string]attr.Value{
				"app_id": cfg.AppId,
			}

			if !verifyFlyioDestinationConfigState(rawFlyioCfg, &diags) {
				return nil, diags
			}

			destinationConfig["appId"] = cfg.AppId.ValueString()

			return destinationConfig, diags
		},
		ReadDestinationConfigFromApi: func(ctx context.Context, secretSync infisical.SecretSync) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			appIdVal, ok := secretSync.DestinationConfig["appId"].(string)
			if !ok {
				diags.AddError(
					"Invalid type",
					"Expected 'appId' to be a string but got something else",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			destinationConfig := map[string]attr.Value{
				"app_id": types.StringValue(appIdVal),
			}

			if !verifyFlyioDestinationConfigState(destinationConfig, &diags) {
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			return types.ObjectValue(map[string]attr.Type{
				"app_id": types.StringType,
			}, destinationConfig)
		},
	}
}
