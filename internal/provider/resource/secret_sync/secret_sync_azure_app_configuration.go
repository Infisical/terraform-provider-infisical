package resource

import (
	"context"
	infisical "terraform-provider-infisical/internal/client"
	infisicalclient "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// SecretSyncAzureAppConfigurationDestinationConfigModel describes the data source data model.
type SecretSyncAzureAppConfigurationDestinationConfigModel struct {
	ConfigurationURL types.String `tfsdk:"configuration_url"`
	Label            types.String `tfsdk:"label"`
}

func NewSecretSyncAzureAppConfigurationResource() resource.Resource {
	return &SecretSyncBaseResource{
		App:              infisical.SecretSyncAppAzureAppConfiguration,
		SyncName:         "Azure App Configuration",
		ResourceTypeName: "_secret_sync_azure_app_configuration",
		AppConnection:    infisical.AppConnectionAppAzure,
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
