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

// SecretSyncAzureDevOpsDestinationConfigModel describes the data source data model.
type SecretSyncAzureDevOpsDestinationConfigModel struct {
	DevopsProjectId types.String `tfsdk:"devops_project_id"`
}

type SecretSyncAzureDevOpsSyncOptionsModel struct {
	DisableSecretDeletion types.Bool   `tfsdk:"disable_secret_deletion"`
	KeySchema             types.String `tfsdk:"key_schema"`
}

func NewSecretSyncAzureDevOpsResource() resource.Resource {
	return &SecretSyncBaseResource{
		App:              infisical.SecretSyncAppAzureDevOps,
		SyncName:         "Azure DevOps",
		ResourceTypeName: "_secret_sync_azure_devops",
		AppConnection:    infisical.AppConnectionAppAzure,
		DestinationConfigAttributes: map[string]schema.Attribute{
			"devops_project_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the Azure DevOps project to sync secrets to.",
			},
		},
		SyncOptionsAttributes: map[string]schema.Attribute{
			"disable_secret_deletion": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "When set to true, Infisical will not remove secrets from Azure DevOps. Enable this option if you intend to manage some secrets manually outside of Infisical.",
				Default:     booldefault.StaticBool(false),
			},
			"key_schema": schema.StringAttribute{
				Optional:    true,
				Description: "The format to use for structuring secret keys in the Azure DevOps destination.",
			},
		},

		ReadSyncOptionsForCreateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			syncOptionsMap := make(map[string]interface{})

			var syncOptions SecretSyncAzureDevOpsSyncOptionsModel
			diags := plan.SyncOptions.As(ctx, &syncOptions, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			syncOptionsMap["initialSyncBehavior"] = "overwrite-destination" // Azure DevOps does not support importing secrets
			syncOptionsMap["disableSecretDeletion"] = syncOptions.DisableSecretDeletion.ValueBool()
			syncOptionsMap["keySchema"] = syncOptions.KeySchema.ValueString()
			return syncOptionsMap, nil

		},

		ReadSyncOptionsForUpdateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel, state SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			syncOptionsMap := make(map[string]interface{})

			var syncOptions SecretSyncAzureDevOpsSyncOptionsModel
			diags := plan.SyncOptions.As(ctx, &syncOptions, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			syncOptionsMap["initialSyncBehavior"] = "overwrite-destination" // Azure DevOps does not support importing secrets
			syncOptionsMap["disableSecretDeletion"] = syncOptions.DisableSecretDeletion.ValueBool()
			syncOptionsMap["keySchema"] = syncOptions.KeySchema.ValueString()
			return syncOptionsMap, nil
		},

		ReadSyncOptionsFromApi: func(ctx context.Context, secretSync infisicalclient.SecretSync) (types.Object, diag.Diagnostics) {

			disableSecretDeletion, ok := secretSync.SyncOptions["disableSecretDeletion"].(bool)
			if !ok {
				disableSecretDeletion = false
			}

			syncOptionsMap := map[string]attr.Value{
				"disable_secret_deletion": types.BoolValue(disableSecretDeletion),
			}

			keySchema, ok := secretSync.SyncOptions["keySchema"].(string)
			if keySchema == "" || !ok {
				syncOptionsMap["key_schema"] = types.StringNull()
			} else {
				syncOptionsMap["key_schema"] = types.StringValue(keySchema)
			}

			return types.ObjectValue(map[string]attr.Type{
				"disable_secret_deletion": types.BoolType,
				"key_schema":              types.StringType,
			}, syncOptionsMap)
		},

		ReadDestinationConfigForCreateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			destinationConfig := make(map[string]interface{})

			var devOpsConfig SecretSyncAzureDevOpsDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &devOpsConfig, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			destinationConfig["devopsProjectId"] = devOpsConfig.DevopsProjectId.ValueString()

			return destinationConfig, diags
		},
		ReadDestinationConfigForUpdateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel, _ SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			destinationConfig := make(map[string]interface{})

			var devOpsConfig SecretSyncAzureDevOpsDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &devOpsConfig, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			destinationConfig["devopsProjectId"] = devOpsConfig.DevopsProjectId.ValueString()

			return destinationConfig, diags
		},
		ReadDestinationConfigFromApi: func(ctx context.Context, secretSync infisicalclient.SecretSync) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			devopsProjectIdVal, ok := secretSync.DestinationConfig["devopsProjectId"].(string)
			if !ok {
				diags.AddError(
					"Invalid Configuration",
					"Expected 'devopsProjectId' to be a string but got something else",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			destinationConfig := map[string]attr.Value{
				"devops_project_id": types.StringValue(devopsProjectIdVal),
			}

			return types.ObjectValue(map[string]attr.Type{
				"devops_project_id": types.StringType,
			}, destinationConfig)
		},
	}
}
