package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type SecretSyncSpaceliftDestinationConfigModel struct {
	ContextId       types.String `tfsdk:"context_id"`
	ContextName     types.String `tfsdk:"context_name"`
	ConfigType      types.String `tfsdk:"config_type"`
	MountPath       types.String `tfsdk:"mount_path"`
	FileMountFormat types.String `tfsdk:"file_mount_format"`
}

type SecretSyncSpaceliftSyncOptionsModel struct {
	DisableSecretDeletion types.Bool   `tfsdk:"disable_secret_deletion"`
	KeySchema             types.String `tfsdk:"key_schema"`
	WriteOnly             types.Bool   `tfsdk:"write_only"`
	IncludeAllSubFolders  types.Bool   `tfsdk:"include_all_sub_folders"`
}

func readSpaceliftSyncOptionsFromPlan(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
	var syncOptions SecretSyncSpaceliftSyncOptionsModel
	diags := plan.SyncOptions.As(ctx, &syncOptions, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, diags
	}

	syncOptionsMap := map[string]interface{}{
		"initialSyncBehavior":   "overwrite-destination", // Spacelift does not support importing secrets
		"disableSecretDeletion": syncOptions.DisableSecretDeletion.ValueBool(),
		"writeOnly":             syncOptions.WriteOnly.ValueBool(),
		"includeAllSubFolders":  syncOptions.IncludeAllSubFolders.ValueBool(),
	}
	if !syncOptions.KeySchema.IsNull() {
		syncOptionsMap["keySchema"] = syncOptions.KeySchema.ValueString()
	}

	return syncOptionsMap, diags
}

func readSpaceliftDestinationConfigFromPlan(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
	var cfg SecretSyncSpaceliftDestinationConfigModel
	diags := plan.DestinationConfig.As(ctx, &cfg, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, diags
	}

	// skips the duplicate destination check in ModifyPlan until all values are known
	for field, value := range plan.DestinationConfig.Attributes() {
		if value.IsUnknown() {
			diags.AddError("Invalid destination config", fmt.Sprintf("Expected '%s' to be known", field))
			return nil, diags
		}
	}

	destinationConfig := map[string]interface{}{
		"contextId":   cfg.ContextId.ValueString(),
		"contextName": cfg.ContextName.ValueString(),
		"configType":  cfg.ConfigType.ValueString(),
	}
	if !cfg.MountPath.IsNull() {
		destinationConfig["mountPath"] = cfg.MountPath.ValueString()
	}
	if !cfg.FileMountFormat.IsNull() {
		destinationConfig["fileMountFormat"] = cfg.FileMountFormat.ValueString()
	}

	return destinationConfig, diags
}

func NewSecretSyncSpaceliftResource() resource.Resource {
	return &SecretSyncBaseResource{
		App:              infisical.SecretSyncAppSpacelift,
		SyncName:         "Spacelift",
		ResourceTypeName: "_secret_sync_spacelift",
		AppConnection:    infisical.AppConnectionAppSpacelift,
		DestinationConfigAttributes: map[string]schema.Attribute{
			"context_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the Spacelift context to sync secrets to.",
			},
			"context_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Spacelift context to sync secrets to.",
			},
			"config_type": schema.StringAttribute{
				Required:    true,
				Description: "The type of config element to create in Spacelift. Supported options: environment-variable, file-mount",
				Validators:  []validator.String{stringvalidator.OneOf("environment-variable", "file-mount")},
			},
			"mount_path": schema.StringAttribute{
				Optional:    true,
				Description: "The file path for the mounted file relative to /mnt/workspace/. Required when config_type is 'file-mount' and file_mount_format is not 'secret-per-file'. When file_mount_format is 'secret-per-file', this is the directory path.",
			},
			"file_mount_format": schema.StringAttribute{
				Optional:    true,
				Description: "The format for file mount config elements. Supported options: dot-env (default), secret-per-file",
				Validators:  []validator.String{stringvalidator.OneOf("dot-env", "secret-per-file")},
			},
		},
		SyncOptionsAttributes: map[string]schema.Attribute{
			"disable_secret_deletion": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "When set to true, Infisical will not remove secrets from Spacelift. Enable this option if you intend to manage some secrets manually outside of Infisical.",
				Default:     booldefault.StaticBool(false),
			},
			"key_schema": schema.StringAttribute{
				Optional:    true,
				Description: "The format to use for structuring secret keys in the Spacelift destination.",
			},
			"write_only": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether secrets should be marked as secret in Spacelift. Secret values are only available to Runs and Tasks and are not accessible in the web GUI or through the API.",
				Default:     booldefault.StaticBool(false),
			},
			"include_all_sub_folders": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to sync secrets from folders beneath the source path as well.",
				Default:     booldefault.StaticBool(false),
			},
		},

		ReadSyncOptionsForCreateFromPlan: readSpaceliftSyncOptionsFromPlan,
		ReadSyncOptionsForUpdateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel, _ SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			return readSpaceliftSyncOptionsFromPlan(ctx, plan)
		},
		ReadSyncOptionsFromApi: func(ctx context.Context, secretSync infisical.SecretSync) (types.Object, diag.Diagnostics) {
			disableSecretDeletion, _ := secretSync.SyncOptions["disableSecretDeletion"].(bool)
			writeOnly, _ := secretSync.SyncOptions["writeOnly"].(bool)
			includeAllSubFolders, _ := secretSync.SyncOptions["includeAllSubFolders"].(bool)

			keySchema := types.StringNull()
			if v, ok := secretSync.SyncOptions["keySchema"].(string); ok {
				keySchema = types.StringValue(v)
			}

			return types.ObjectValue(map[string]attr.Type{
				"disable_secret_deletion": types.BoolType,
				"key_schema":              types.StringType,
				"write_only":              types.BoolType,
				"include_all_sub_folders": types.BoolType,
			}, map[string]attr.Value{
				"disable_secret_deletion": types.BoolValue(disableSecretDeletion),
				"key_schema":              keySchema,
				"write_only":              types.BoolValue(writeOnly),
				"include_all_sub_folders": types.BoolValue(includeAllSubFolders),
			})
		},

		ReadDestinationConfigForCreateFromPlan: readSpaceliftDestinationConfigFromPlan,
		ReadDestinationConfigForUpdateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel, _ SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			return readSpaceliftDestinationConfigFromPlan(ctx, plan)
		},
		ReadDestinationConfigFromApi: func(ctx context.Context, secretSync infisical.SecretSync) (types.Object, diag.Diagnostics) {
			stringFromApi := func(key string) types.String {
				if v, ok := secretSync.DestinationConfig[key].(string); ok {
					return types.StringValue(v)
				}
				return types.StringNull()
			}

			return types.ObjectValue(map[string]attr.Type{
				"context_id":        types.StringType,
				"context_name":      types.StringType,
				"config_type":       types.StringType,
				"mount_path":        types.StringType,
				"file_mount_format": types.StringType,
			}, map[string]attr.Value{
				"context_id":        stringFromApi("contextId"),
				"context_name":      stringFromApi("contextName"),
				"config_type":       stringFromApi("configType"),
				"mount_path":        stringFromApi("mountPath"),
				"file_mount_format": stringFromApi("fileMountFormat"),
			})
		},
	}
}
