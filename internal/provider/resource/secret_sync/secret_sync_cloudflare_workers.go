package resource

import (
	"context"
	infisical "terraform-provider-infisical/internal/client"
	infisicalclient "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// SecretSyncCloudflareWorkersDestinationConfigModel describes the data source data model.
type SecretSyncCloudflareWorkersDestinationConfigModel struct {
	ScriptId types.String `tfsdk:"script_id"`
}

type SecretSyncCloudflareWorkersSyncOptionsModel struct {
	InitialSyncBehavior   types.String `tfsdk:"initial_sync_behavior"`
	DisableSecretDeletion types.Bool   `tfsdk:"disable_secret_deletion"`
	KeySchema             types.String `tfsdk:"key_schema"`
}

func NewSecretSyncCloudflareWorkersResource() resource.Resource {
	return &SecretSyncBaseResource{
		App:              infisical.SecretSyncAppCloudflareWorkers,
		SyncName:         "Cloudflare Workers",
		ResourceTypeName: "_secret_sync_cloudflare_workers",
		AppConnection:    infisical.AppConnectionAppCloudflare,
		DestinationConfigAttributes: map[string]schema.Attribute{
			"script_id": schema.StringAttribute{
				Required:    true,
				Description: "The Cloudflare Workers script ID where the secrets will be synced",
			},
		},
		SyncOptionsAttributes: map[string]schema.Attribute{
			"initial_sync_behavior": schema.StringAttribute{
				Required:    true,
				Description: "Specify how Infisical should resolve the initial sync to the destination. Supported options: overwrite-destination, import-prioritize-source, import-prioritize-destination",
			},
			"disable_secret_deletion": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "When set to true, Infisical will not remove secrets from Cloudflare Workers. Enable this option if you intend to manage some secrets manually outside of Infisical.",
				Default:     booldefault.StaticBool(false),
			},
			"key_schema": schema.StringAttribute{
				Optional:    true,
				Description: "The format to use for structuring secret keys in the Cloudflare Workers destination.",
			},
		},

		ReadSyncOptionsForCreateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			syncOptionsMap := make(map[string]interface{})

			var syncOptions SecretSyncCloudflareWorkersSyncOptionsModel
			diags := plan.SyncOptions.As(ctx, &syncOptions, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if syncOptions.InitialSyncBehavior.IsNull() || syncOptions.InitialSyncBehavior.ValueString() == "" {
				diags.AddError(
					"Unable to create Cloudflare Workers secret sync",
					"Initial sync behavior must be specified",
				)
				return nil, diags
			}

			syncOptionsMap["initialSyncBehavior"] = syncOptions.InitialSyncBehavior.ValueString()

			if !syncOptions.DisableSecretDeletion.IsNull() {
				syncOptionsMap["disableSecretDeletion"] = syncOptions.DisableSecretDeletion.ValueBool()
			}

			syncOptionsMap["keySchema"] = syncOptions.KeySchema.ValueString()

			return syncOptionsMap, diags
		},

		ReadSyncOptionsForUpdateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel, state SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			syncOptionsMap := make(map[string]interface{})

			var syncOptionsFromPlan SecretSyncCloudflareWorkersSyncOptionsModel
			diags := plan.SyncOptions.As(ctx, &syncOptionsFromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var syncOptionsFromState SecretSyncCloudflareWorkersSyncOptionsModel
			diags = state.SyncOptions.As(ctx, &syncOptionsFromState, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if syncOptionsFromPlan.InitialSyncBehavior.IsUnknown() {
				syncOptionsMap["initialSyncBehavior"] = syncOptionsFromState.InitialSyncBehavior.ValueString()
			} else {
				if syncOptionsFromPlan.InitialSyncBehavior.IsNull() || syncOptionsFromPlan.InitialSyncBehavior.ValueString() == "" {
					diags.AddError(
						"Unable to update Cloudflare Workers secret sync",
						"Initial sync behavior must be specified",
					)
					return nil, diags
				}
				syncOptionsMap["initialSyncBehavior"] = syncOptionsFromPlan.InitialSyncBehavior.ValueString()
			}

			if syncOptionsFromPlan.DisableSecretDeletion.IsUnknown() {
				syncOptionsMap["disableSecretDeletion"] = syncOptionsFromState.DisableSecretDeletion.ValueBool()
			} else {
				syncOptionsMap["disableSecretDeletion"] = syncOptionsFromPlan.DisableSecretDeletion.ValueBool()
			}

			syncOptionsMap["keySchema"] = syncOptionsFromPlan.KeySchema.ValueString()

			return syncOptionsMap, diags
		},

		ReadSyncOptionsFromApi: func(ctx context.Context, secretSync infisicalclient.SecretSync) (types.Object, diag.Diagnostics) {
			syncOptionsAttrTypes := map[string]attr.Type{
				"initial_sync_behavior":   types.StringType,
				"disable_secret_deletion": types.BoolType,
				"key_schema":              types.StringType,
			}

			diags := diag.Diagnostics{}

			initialSyncBehaviorVal, ok := secretSync.SyncOptions["initialSyncBehavior"].(string)
			if !ok {
				diags.AddError(
					"Invalid initial sync behavior type",
					"Expected 'initialSyncBehavior' to be a string but got something else",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			disableSecretDeletionVal, ok := secretSync.SyncOptions["disableSecretDeletion"].(bool)
			if !ok {
				// Default to false if not present
				disableSecretDeletionVal = false
			}

			syncOptionsAttrValues := map[string]attr.Value{
				"initial_sync_behavior":   types.StringValue(initialSyncBehaviorVal),
				"disable_secret_deletion": types.BoolValue(disableSecretDeletionVal),
			}

			keySchema, ok := secretSync.SyncOptions["keySchema"].(string)
			if keySchema == "" || !ok {
				syncOptionsAttrValues["key_schema"] = types.StringNull()
			} else {
				syncOptionsAttrValues["key_schema"] = types.StringValue(keySchema)
			}

			return types.ObjectValue(syncOptionsAttrTypes, syncOptionsAttrValues)
		},

		ReadDestinationConfigForCreateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			destinationConfigMap := make(map[string]interface{})

			var destinationConfig SecretSyncCloudflareWorkersDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &destinationConfig, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if destinationConfig.ScriptId.IsNull() || destinationConfig.ScriptId.ValueString() == "" {
				diags.AddError(
					"Unable to create Cloudflare Workers secret sync",
					"Script ID must be specified",
				)
				return nil, diags
			}

			destinationConfigMap["scriptId"] = destinationConfig.ScriptId.ValueString()

			return destinationConfigMap, diags
		},

		ReadDestinationConfigForUpdateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel, state SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			destinationConfigMap := make(map[string]interface{})

			var destinationConfigFromPlan SecretSyncCloudflareWorkersDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &destinationConfigFromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var destinationConfigFromState SecretSyncCloudflareWorkersDestinationConfigModel
			diags = state.DestinationConfig.As(ctx, &destinationConfigFromState, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if destinationConfigFromPlan.ScriptId.IsUnknown() {
				destinationConfigMap["scriptId"] = destinationConfigFromState.ScriptId.ValueString()
			} else {
				if destinationConfigFromPlan.ScriptId.IsNull() || destinationConfigFromPlan.ScriptId.ValueString() == "" {
					diags.AddError(
						"Unable to update Cloudflare Workers secret sync",
						"Script ID must be specified",
					)
					return nil, diags
				}
				destinationConfigMap["scriptId"] = destinationConfigFromPlan.ScriptId.ValueString()
			}

			return destinationConfigMap, diags
		},

		ReadDestinationConfigFromApi: func(ctx context.Context, secretSync infisicalclient.SecretSync) (types.Object, diag.Diagnostics) {
			destinationConfigAttrTypes := map[string]attr.Type{
				"script_id": types.StringType,
			}

			scriptIdVal, ok := secretSync.DestinationConfig["scriptId"].(string)
			if !ok {
				diags := diag.Diagnostics{}
				diags.AddError(
					"Invalid script ID type",
					"Expected 'scriptId' to be a string but got something else",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			destinationConfigAttrValues := map[string]attr.Value{
				"script_id": types.StringValue(scriptIdVal),
			}

			return types.ObjectValue(destinationConfigAttrTypes, destinationConfigAttrValues)
		},
	}
}
