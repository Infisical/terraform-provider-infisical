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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// SecretSyncAzureAppConfigurationDestinationConfigModel describes the data source data model.
type SecretSyncAzureAppConfigurationDestinationConfigModel struct {
	ConfigurationURL types.String `tfsdk:"configuration_url"`
	Label            types.String `tfsdk:"label"`
}

type SecretSyncAzureAppConfigurationSyncOptionsModel struct {
	InitialSyncBehavior   types.String `tfsdk:"initial_sync_behavior"`
	DisableSecretDeletion types.Bool   `tfsdk:"disable_secret_deletion"`
	KeySchema             types.String `tfsdk:"key_schema"`
}

func NewSecretSyncAzureAppConfigurationResource() resource.Resource {
	return &SecretSyncBaseResource{
		App:              infisical.SecretSyncAppAzureAppConfiguration,
		SyncName:         "Azure App Configuration",
		ResourceTypeName: "_secret_sync_azure_app_configuration",
		AppConnection:    infisical.AppConnectionAppAzure,
		SyncOptionsAttributes: map[string]schema.Attribute{
			"initial_sync_behavior": schema.StringAttribute{
				Required:    true,
				Description: "Specify how Infisical should resolve the initial sync to the destination. Supported options: overwrite-destination, import-prioritize-source, import-prioritize-destination",
			},
			"disable_secret_deletion": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "When set to true, Infisical will not remove secrets from Azure App Configuration. Enable this option if you intend to manage some secrets manually outside of Infisical.",
				Default:     booldefault.StaticBool(false),
			},
			"key_schema": schema.StringAttribute{
				Optional:    true,
				Description: "The format to use for structuring secret keys in the Azure App Configuration destination.",
			},
		},

		ReadSyncOptionsForCreateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			syncOptionsMap := make(map[string]interface{})

			var syncOptions SecretSyncAzureAppConfigurationSyncOptionsModel
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

			var syncOptions SecretSyncAzureAppConfigurationSyncOptionsModel
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
		DestinationConfigAttributes: map[string]schema.Attribute{
			"configuration_url": schema.StringAttribute{
				Required:    true,
				Description: "The URL of your Azure App Configuration",
			},

			// "" becomes null, lets use a planmodifer to handle this
			"label": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The label to attach to secrets created in Azure App Configuration",
				Default:     stringdefault.StaticString(""),
			},
		},
		ReadDestinationConfigForCreateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			destinationConfig := make(map[string]interface{})

			var appConfigurationConfig SecretSyncAzureAppConfigurationDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &appConfigurationConfig, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			destinationConfig["configurationUrl"] = appConfigurationConfig.ConfigurationURL.ValueString()
			destinationConfig["label"] = appConfigurationConfig.Label.ValueString()

			return destinationConfig, diags
		},
		ReadDestinationConfigForUpdateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel, _ SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			destinationConfig := make(map[string]interface{})

			var appConfigurationConfig SecretSyncAzureAppConfigurationDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &appConfigurationConfig, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			destinationConfig["configurationUrl"] = appConfigurationConfig.ConfigurationURL.ValueString()
			destinationConfig["label"] = appConfigurationConfig.Label.ValueString()

			return destinationConfig, diags
		},
		ReadDestinationConfigFromApi: func(ctx context.Context, secretSync infisicalclient.SecretSync) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			urlVal, ok := secretSync.DestinationConfig["configurationUrl"].(string)
			if !ok {
				diags.AddError(
					"Invalid configuration URL type",
					"Expected 'configuration_url' to be a string but got something else",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			labelVal, ok := secretSync.DestinationConfig["label"].(string)
			if !ok {
				diags.AddError(
					"Invalid label type",
					"Expected 'label' to be a string but got something else",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			destinationConfig := map[string]attr.Value{
				"configuration_url": types.StringValue(urlVal),
				"label":             types.StringValue(labelVal),
			}

			return types.ObjectValue(map[string]attr.Type{
				"configuration_url": types.StringType,
				"label":             types.StringType,
			}, destinationConfig)
		},
	}
}
