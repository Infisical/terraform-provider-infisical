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

// SecretSyncCloudflarePagesDestinationConfigModel describes the data source data model.
type SecretSyncCloudflarePagesDestinationConfigModel struct {
	ProjectName types.String `tfsdk:"project_name"`
	Environment types.String `tfsdk:"environment"`
}

type SecretSyncCloudflarePagesSyncOptionsModel struct {
	InitialSyncBehavior   types.String `tfsdk:"initial_sync_behavior"`
	DisableSecretDeletion types.Bool   `tfsdk:"disable_secret_deletion"`
	KeySchema             types.String `tfsdk:"key_schema"`
}

func NewSecretSyncCloudflarePagesResource() resource.Resource {
	return &SecretSyncBaseResource{
		App:              infisical.SecretSyncAppCloudflarePages,
		SyncName:         "Cloudflare Pages",
		ResourceTypeName: "_secret_sync_cloudflare_pages",
		AppConnection:    infisical.AppConnectionAppCloudflare,
		DestinationConfigAttributes: map[string]schema.Attribute{
			"project_name": schema.StringAttribute{
				Required:    true,
				Description: "The Cloudflare Pages project name where the secrets will be synced",
			},
			"environment": schema.StringAttribute{
				Required:    true,
				Description: "The Cloudflare Pages environment (production, preview) where the secrets will be synced",
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
				Description: "When set to true, Infisical will not remove secrets from Cloudflare Pages. Enable this option if you intend to manage some secrets manually outside of Infisical.",
				Default:     booldefault.StaticBool(false),
			},
			"key_schema": schema.StringAttribute{
				Optional:    true,
				Description: "The format to use for structuring secret keys in the Cloudflare Pages destination.",
			},
		},

		ReadSyncOptionsForCreateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			syncOptionsMap := make(map[string]interface{})

			var syncOptions SecretSyncCloudflarePagesSyncOptionsModel
			diags := plan.SyncOptions.As(ctx, &syncOptions, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if syncOptions.InitialSyncBehavior.IsNull() || syncOptions.InitialSyncBehavior.ValueString() == "" {
				diags.AddError(
					"Unable to create Cloudflare Pages secret sync",
					"Initial sync behavior must be specified",
				)
				return nil, diags
			}

			syncOptionsMap["initialSyncBehavior"] = syncOptions.InitialSyncBehavior.ValueString()

			if !syncOptions.DisableSecretDeletion.IsNull() {
				syncOptionsMap["disableSecretDeletion"] = syncOptions.DisableSecretDeletion.ValueBool()
			}

			if !syncOptions.KeySchema.IsNull() && syncOptions.KeySchema.ValueString() != "" {
				syncOptionsMap["keySchema"] = syncOptions.KeySchema.ValueString()
			}

			return syncOptionsMap, diags
		},

		ReadSyncOptionsForUpdateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel, state SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			syncOptionsMap := make(map[string]interface{})

			var syncOptionsFromPlan SecretSyncCloudflarePagesSyncOptionsModel
			diags := plan.SyncOptions.As(ctx, &syncOptionsFromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var syncOptionsFromState SecretSyncCloudflarePagesSyncOptionsModel
			diags = state.SyncOptions.As(ctx, &syncOptionsFromState, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if syncOptionsFromPlan.InitialSyncBehavior.IsUnknown() {
				syncOptionsMap["initialSyncBehavior"] = syncOptionsFromState.InitialSyncBehavior.ValueString()
			} else {
				if syncOptionsFromPlan.InitialSyncBehavior.IsNull() || syncOptionsFromPlan.InitialSyncBehavior.ValueString() == "" {
					diags.AddError(
						"Unable to update Cloudflare Pages secret sync",
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

			if syncOptionsFromPlan.KeySchema.IsUnknown() {
				if !syncOptionsFromState.KeySchema.IsNull() {
					syncOptionsMap["keySchema"] = syncOptionsFromState.KeySchema.ValueString()
				}
			} else {
				if !syncOptionsFromPlan.KeySchema.IsNull() && syncOptionsFromPlan.KeySchema.ValueString() != "" {
					syncOptionsMap["keySchema"] = syncOptionsFromPlan.KeySchema.ValueString()
				}
			}

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

			keySchemaVal := ""
			if keySchemaRaw, exists := secretSync.SyncOptions["keySchema"]; exists {
				if keySchemaStr, ok := keySchemaRaw.(string); ok {
					keySchemaVal = keySchemaStr
				}
			}

			syncOptionsAttrValues := map[string]attr.Value{
				"initial_sync_behavior":   types.StringValue(initialSyncBehaviorVal),
				"disable_secret_deletion": types.BoolValue(disableSecretDeletionVal),
				"key_schema":              types.StringValue(keySchemaVal),
			}

			return types.ObjectValue(syncOptionsAttrTypes, syncOptionsAttrValues)
		},

		ReadDestinationConfigForCreateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			destinationConfigMap := make(map[string]interface{})

			var destinationConfig SecretSyncCloudflarePagesDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &destinationConfig, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if destinationConfig.ProjectName.IsNull() || destinationConfig.ProjectName.ValueString() == "" {
				diags.AddError(
					"Unable to create Cloudflare Pages secret sync",
					"Project name must be specified",
				)
				return nil, diags
			}

			if destinationConfig.Environment.IsNull() || destinationConfig.Environment.ValueString() == "" {
				diags.AddError(
					"Unable to create Cloudflare Pages secret sync",
					"Environment must be specified",
				)
				return nil, diags
			}

			destinationConfigMap["projectName"] = destinationConfig.ProjectName.ValueString()
			destinationConfigMap["environment"] = destinationConfig.Environment.ValueString()

			return destinationConfigMap, diags
		},

		ReadDestinationConfigForUpdateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel, state SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			destinationConfigMap := make(map[string]interface{})

			var destinationConfigFromPlan SecretSyncCloudflarePagesDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &destinationConfigFromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var destinationConfigFromState SecretSyncCloudflarePagesDestinationConfigModel
			diags = state.DestinationConfig.As(ctx, &destinationConfigFromState, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if destinationConfigFromPlan.ProjectName.IsUnknown() {
				destinationConfigMap["projectName"] = destinationConfigFromState.ProjectName.ValueString()
			} else {
				if destinationConfigFromPlan.ProjectName.IsNull() || destinationConfigFromPlan.ProjectName.ValueString() == "" {
					diags.AddError(
						"Unable to update Cloudflare Pages secret sync",
						"Project name must be specified",
					)
					return nil, diags
				}
				destinationConfigMap["projectName"] = destinationConfigFromPlan.ProjectName.ValueString()
			}

			if destinationConfigFromPlan.Environment.IsUnknown() {
				destinationConfigMap["environment"] = destinationConfigFromState.Environment.ValueString()
			} else {
				if destinationConfigFromPlan.Environment.IsNull() || destinationConfigFromPlan.Environment.ValueString() == "" {
					diags.AddError(
						"Unable to update Cloudflare Pages secret sync",
						"Environment must be specified",
					)
					return nil, diags
				}
				destinationConfigMap["environment"] = destinationConfigFromPlan.Environment.ValueString()
			}

			return destinationConfigMap, diags
		},

		ReadDestinationConfigFromApi: func(ctx context.Context, secretSync infisicalclient.SecretSync) (types.Object, diag.Diagnostics) {
			destinationConfigAttrTypes := map[string]attr.Type{
				"project_name": types.StringType,
				"environment":  types.StringType,
			}

			diags := diag.Diagnostics{}

			projectNameVal, ok := secretSync.DestinationConfig["projectName"].(string)
			if !ok {
				diags.AddError(
					"Invalid project name type",
					"Expected 'projectName' to be a string but got something else",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			environmentVal, ok := secretSync.DestinationConfig["environment"].(string)
			if !ok {
				diags.AddError(
					"Invalid environment type",
					"Expected 'environment' to be a string but got something else",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			destinationConfigAttrValues := map[string]attr.Value{
				"project_name": types.StringValue(projectNameVal),
				"environment":  types.StringValue(environmentVal),
			}

			return types.ObjectValue(destinationConfigAttrTypes, destinationConfigAttrValues)
		},
	}
}
