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

// SecretSync1PasswordDestinationConfigModel describes the data source data model.
type SecretSync1PasswordDestinationConfigModel struct {
	ValueLabel types.String `tfsdk:"value_label"` // optional
	VaultId    types.String `tfsdk:"vault_id"`    // required
}

type SecretSync1PasswordSyncOptionsModel struct {
	InitialSyncBehavior   types.String `tfsdk:"initial_sync_behavior"`
	DisableSecretDeletion types.Bool   `tfsdk:"disable_secret_deletion"`
	KeySchema             types.String `tfsdk:"key_schema"`
}

func NewSecretSync1PasswordResource() resource.Resource {
	return &SecretSyncBaseResource{
		App:              infisical.SecretSyncApp1Password,
		SyncName:         "1Password",
		ResourceTypeName: "_secret_sync_1password",
		AppConnection:    infisical.AppConnectionApp1Password,
		DestinationConfigAttributes: map[string]schema.Attribute{
			"vault_id": schema.StringAttribute{
				Required:    true,
				Description: "The The ID of the 1Password vault to sync secrets to",
			},
			"value_label": schema.StringAttribute{
				Optional:    true,
				Description: "The label of the 1Password item field which will hold your secret value. For example, if you were to sync Infisical secret 'foo: bar', the 1Password item equivalent would have an item title of 'foo', and a field on that item 'value: bar'. The field label 'value' is what gets changed by this option",
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
				Description: "When set to true, Infisical will not remove secrets from 1Password. Enable this option if you intend to manage some secrets manually outside of Infisical.",
				Default:     booldefault.StaticBool(false),
			},
			"key_schema": schema.StringAttribute{
				Optional:    true,
				Description: "The format to use for structuring secret keys in the 1Password destination.",
			},
		},

		ReadSyncOptionsForCreateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			syncOptionsMap := make(map[string]interface{})

			var syncOptions SecretSync1PasswordSyncOptionsModel
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

			var syncOptions SecretSync1PasswordSyncOptionsModel
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

		ReadDestinationConfigForCreateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			destinationConfig := make(map[string]interface{})

			var keyVaultConfig SecretSync1PasswordDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &keyVaultConfig, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if keyVaultConfig.VaultId.IsNull() || keyVaultConfig.VaultId.ValueString() == "" {
				diags.AddError(
					"Invalid Vault ID",
					"Expected 'vault_id' to be a string but got something else",
				)
				return nil, diags
			}

			destinationConfig["vaultId"] = keyVaultConfig.VaultId.ValueString()

			if !keyVaultConfig.ValueLabel.IsNull() && keyVaultConfig.ValueLabel.ValueString() != "" {
				destinationConfig["valueLabel"] = keyVaultConfig.ValueLabel.ValueString()
			} else {
				destinationConfig["valueLabel"] = ""
			}

			return destinationConfig, diags
		},
		ReadDestinationConfigForUpdateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel, _ SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			destinationConfig := make(map[string]interface{})

			var keyVaultConfig SecretSync1PasswordDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &keyVaultConfig, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if keyVaultConfig.VaultId.IsNull() || keyVaultConfig.VaultId.ValueString() == "" {
				diags.AddError(
					"Invalid Vault ID",
					"Expected 'vault_id' to be a string but got something else",
				)
				return nil, diags
			}

			destinationConfig["vaultId"] = keyVaultConfig.VaultId.ValueString()

			if !keyVaultConfig.ValueLabel.IsNull() && keyVaultConfig.ValueLabel.ValueString() != "" {
				destinationConfig["valueLabel"] = keyVaultConfig.ValueLabel.ValueString()
			} else {
				destinationConfig["valueLabel"] = ""
			}

			return destinationConfig, diags
		},
		ReadDestinationConfigFromApi: func(ctx context.Context, secretSync infisicalclient.SecretSync) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			destinationConfig := map[string]attr.Value{
				"vault_id":    types.StringNull(),
				"value_label": types.StringValue(""),
			}

			vaultIdVal, ok := secretSync.DestinationConfig["vaultId"].(string)
			if !ok {
				diags.AddError(
					"Invalid Vault ID",
					"Expected 'vault_id' to be a string but got something else",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			destinationConfig["vault_id"] = types.StringValue(vaultIdVal)

			valueLabelVal, ok := secretSync.DestinationConfig["valueLabel"].(string)
			if ok && valueLabelVal != "" {
				destinationConfig["value_label"] = types.StringValue(valueLabelVal)
			}

			return types.ObjectValue(map[string]attr.Type{
				"vault_id":    types.StringType,
				"value_label": types.StringType,
			}, destinationConfig)
		},
	}
}
