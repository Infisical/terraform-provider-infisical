package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"
	infisicalclient "terraform-provider-infisical/internal/client"
	"terraform-provider-infisical/internal/pkg/terraform"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func verifyDatabricksDestinationConfigState(destinationConfig map[string]attr.Value, diags *diag.Diagnostics) bool {
	scopeAttr, exists := destinationConfig["scope"]
	if !exists {
		diags.AddError("Invalid destination config", "Expected 'scope' to be present")
		return false
	}

	scopeVal, ok := scopeAttr.(types.String)
	if !ok {
		diags.AddError("Invalid destination config", "Expected 'scope' to be a string type")
		return false
	}

	if scopeVal.IsNull() || scopeVal.IsUnknown() {
		diags.AddError("Invalid destination config", "Expected 'scope' to have a value")
		return false
	}

	requiredFields := []string{"scope"}

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

type SecretSyncDatabricksDestinationConfigModel struct {
	Scope types.String `tfsdk:"scope"`
}

type SecretSyncDatabricksSyncOptionsModel struct {
	InitialSyncBehavior   types.String `tfsdk:"initial_sync_behavior"`
	DisableSecretDeletion types.Bool   `tfsdk:"disable_secret_deletion"`
	KeySchema             types.String `tfsdk:"key_schema"`
}

func NewSecretSyncDatabricksResource() resource.Resource {
	return &SecretSyncBaseResource{
		App:              infisical.SecretSyncAppDatabricks,
		SyncName:         "Databricks",
		ResourceTypeName: "_secret_sync_databricks",
		AppConnection:    infisical.AppConnectionAppDatabricks,
		DestinationConfigAttributes: map[string]schema.Attribute{
			"scope": schema.StringAttribute{
				Required:    true,
				Description: "The Databricks secret scope to sync secrets to.",
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
				Description: "When set to true, Infisical will not remove secrets from Databricks. Enable this option if you intend to manage some secrets manually outside of Infisical.",
				Default:     booldefault.StaticBool(false),
			},
			"key_schema": schema.StringAttribute{
				Optional:    true,
				Description: "The format to use for structuring secret keys in the Databricks destination.",
			},
		},

		ReadSyncOptionsForCreateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			syncOptionsMap := make(map[string]interface{})

			var syncOptions SecretSyncDatabricksSyncOptionsModel
			diags := plan.SyncOptions.As(ctx, &syncOptions, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			syncOptionsMap["initialSyncBehavior"] = syncOptions.InitialSyncBehavior.ValueString()
			syncOptionsMap["disableSecretDeletion"] = syncOptions.DisableSecretDeletion.ValueBool()
			syncOptionsMap["keySchema"] = syncOptions.KeySchema.ValueString()

			return syncOptionsMap, nil
		},

		ReadSyncOptionsFromApi: func(ctx context.Context, secretSync infisicalclient.SecretSync) (types.Object, diag.Diagnostics) {
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

			var syncOptions SecretSyncDatabricksSyncOptionsModel
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

			var cfg SecretSyncDatabricksDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &cfg, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			rawDatabricksCfg := map[string]attr.Value{
				"scope": cfg.Scope,
			}

			if !verifyDatabricksDestinationConfigState(rawDatabricksCfg, &diags) {
				return nil, diags
			}

			destinationConfig["scope"] = cfg.Scope.ValueString()

			return destinationConfig, diags
		},
		ReadDestinationConfigForUpdateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel, _ SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			destinationConfig := make(map[string]interface{})

			var cfg SecretSyncDatabricksDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &cfg, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			rawDatabricksCfg := map[string]attr.Value{
				"scope": cfg.Scope,
			}

			if !verifyDatabricksDestinationConfigState(rawDatabricksCfg, &diags) {
				return nil, diags
			}

			destinationConfig["scope"] = cfg.Scope.ValueString()

			return destinationConfig, diags
		},
		ReadDestinationConfigFromApi: func(ctx context.Context, secretSync infisicalclient.SecretSync) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			scopeVal, ok := secretSync.DestinationConfig["scope"].(string)
			if !ok {
				diags.AddError(
					"Invalid type",
					"Expected 'scope' to be a string but got something else",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			destinationConfig := map[string]attr.Value{
				"scope": types.StringValue(scopeVal),
			}

			if !verifyDatabricksDestinationConfigState(destinationConfig, &diags) {
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			return types.ObjectValue(map[string]attr.Type{
				"scope": types.StringType,
			}, destinationConfig)
		},
	}
}
