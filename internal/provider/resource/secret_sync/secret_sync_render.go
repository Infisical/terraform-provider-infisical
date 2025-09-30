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

type RenderSyncScope string

type RenderSyncType string

const (
	RenderSyncTypeEnv  RenderSyncType = "env"
	RenderSyncTypeFile RenderSyncType = "file"
)

const (
	RenderSyncScopeService RenderSyncScope = "service"
)

func verifyRenderDestinationConfigState(destinationConfig map[string]attr.Value, diags *diag.Diagnostics) bool {
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

	scope := RenderSyncScope(scopeVal.ValueString())

	var requiredFields []string

	switch scope {
	case "service":
		requiredFields = []string{"scope", "service_id", "type"}
	default:
		diags.AddError("Invalid config", fmt.Sprintf("Invalid scope '%s' expected options 'service'", scope))
		return false
	}

	typeAttr, exists := destinationConfig["type"]
	if !exists {
		diags.AddError("Invalid destination config", "Expected 'type' to be present")
		return false
	}

	typeVal, ok := typeAttr.(types.String)
	if !ok {
		diags.AddError("Invalid destination config", "Expected 'type' to be a string type")
		return false
	}

	if typeVal.IsNull() || typeVal.IsUnknown() {
		diags.AddError("Invalid destination config", "Expected 'type' to have a value")
		return false
	}

	syncType := RenderSyncType(typeVal.ValueString())

	if syncType != RenderSyncTypeEnv && syncType != RenderSyncTypeFile {
		diags.AddError("Invalid destination config", fmt.Sprintf("Invalid type '%s' expected options 'env', 'file'", syncType))
	}

	// Check required fields are not empty
	for _, field := range requiredFields {
		value, exists := destinationConfig[field]
		if !exists {
			diags.AddError("Invalid destination config", fmt.Sprintf("Expected '%s' to be present", field))
			return false
		}

		// Check if the value is null, unknown, or empty based on its type
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

			diags.AddError("Invalid destination config", fmt.Sprintf("Unexpected field '%s' for scope '%s'. Supported destination_config fields are: %v", field, scope, requiredFields))
			return false
		}
	}

	return true
}

// SecretSyncRenderDestinationConfigModel describes the data source data model.

type SecretSyncRenderDestinationConfigModel struct {
	Scope     types.String `tfsdk:"scope"` // service
	Type      types.String `tfsdk:"type"`  // env
	ServiceID types.String `tfsdk:"service_id"`
}

type SecretSyncRenderSyncOptionsModel struct {
	InitialSyncBehavior   types.String `tfsdk:"initial_sync_behavior"`
	DisableSecretDeletion types.Bool   `tfsdk:"disable_secret_deletion"`
	KeySchema             types.String `tfsdk:"key_schema"`
}

func NewSecretSyncRenderResource() resource.Resource {
	return &SecretSyncBaseResource{
		CrossplaneCompatible: false,
		App:                  infisical.SecretSyncAppRender,
		SyncName:             "Render",
		ResourceTypeName:     "_secret_sync_render",
		AppConnection:        infisical.AppConnectionAppRender,
		DestinationConfigAttributes: map[string]schema.Attribute{
			"scope": schema.StringAttribute{
				Required:    true,
				Description: "The Render scope that secrets should be synced to. Supported options: service",
			},
			"service_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the Render service to sync secrets to.",
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "The Render resource type to sync secrets to. Supported options: env, file",
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
				Description: "When set to true, Infisical will not remove secrets from Render. Enable this option if you intend to manage some secrets manually outside of Infisical.",
				Default:     booldefault.StaticBool(false),
			},
			"key_schema": schema.StringAttribute{
				Optional:    true,
				Description: "The format to use for structuring secret keys in the Render destination.",
			},
		},

		ReadSyncOptionsForCreateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			syncOptionsMap := make(map[string]interface{})

			var syncOptions SecretSyncRenderSyncOptionsModel
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

			var syncOptions SecretSyncRenderSyncOptionsModel
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

			var cfg SecretSyncRenderDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &cfg, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			rawRenderCfg := map[string]attr.Value{
				"scope":      cfg.Scope,
				"type":       cfg.Type,
				"service_id": cfg.ServiceID,
			}

			if !verifyRenderDestinationConfigState(rawRenderCfg, &diags) {
				return nil, diags
			}

			destinationConfig["type"] = cfg.Type.ValueString()
			destinationConfig["scope"] = cfg.Scope.ValueString()

			if RenderSyncScope(cfg.Scope.ValueString()) == RenderSyncScopeService {
				destinationConfig["serviceId"] = cfg.ServiceID.ValueString()
			}

			return destinationConfig, diags
		},
		ReadDestinationConfigForUpdateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel, _ SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			destinationConfig := make(map[string]interface{})

			var cfg SecretSyncRenderDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &cfg, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			rawRenderCfg := map[string]attr.Value{
				"scope":      cfg.Scope,
				"service_id": cfg.ServiceID,
				"type":       cfg.Type,
			}

			if !verifyRenderDestinationConfigState(rawRenderCfg, &diags) {
				return nil, diags
			}

			destinationConfig["scope"] = cfg.Scope.ValueString()
			destinationConfig["type"] = cfg.Type.ValueString()

			if RenderSyncScope(cfg.Scope.ValueString()) == RenderSyncScopeService {
				destinationConfig["serviceId"] = cfg.ServiceID.ValueString()
			}

			return destinationConfig, diags
		},
		ReadDestinationConfigFromApi: func(ctx context.Context, secretSync infisical.SecretSync) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			scopeVal, ok := secretSync.DestinationConfig["scope"].(string)
			if !ok {
				diags.AddError(
					"Invalid type",
					"Expected 'scope' to be a string but got something else",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			typeVal, ok := secretSync.DestinationConfig["type"].(string)
			if !ok {
				diags.AddError(
					"Invalid type",
					"Expected 'type' to be a string but got something else",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			destinationConfig := map[string]attr.Value{
				"scope":      types.StringValue(scopeVal),
				"type":       types.StringValue(typeVal),
				"service_id": types.StringNull(),
			}

			if RenderSyncScope(scopeVal) == RenderSyncScopeService {
				serviceIdVal, ok := secretSync.DestinationConfig["serviceId"].(string)
				if !ok {
					diags.AddError(
						"Invalid service ID type",
						"Expected 'serviceId' to be a string but got something else",
					)
					return types.ObjectNull(map[string]attr.Type{}), diags
				}

				destinationConfig["service_id"] = types.StringValue(serviceIdVal)
			}

			if !verifyRenderDestinationConfigState(destinationConfig, &diags) {
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			return types.ObjectValue(map[string]attr.Type{
				"scope":      types.StringType,
				"type":       types.StringType,
				"service_id": types.StringType,
			}, destinationConfig)
		},
	}
}
