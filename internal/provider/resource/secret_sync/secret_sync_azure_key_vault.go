package resource

import (
	"context"
	infisical "terraform-provider-infisical/internal/client"
	infisicalclient "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// SecretSyncAzureKeyVaultDestinationConfigModel describes the data source data model.
type SecretSyncAzureKeyVaultDestinationConfigModel struct {
	VaultBaseURL types.String `tfsdk:"vault_base_url"`
}

func NewSecretSyncAzureKeyVaultResource() resource.Resource {
	return &SecretSyncBaseResource{
		App:              infisical.SecretSyncAppAzureKeyVault,
		SyncName:         "Azure Key Vault",
		ResourceTypeName: "_secret_sync_azure_key_vault",
		AppConnection:    infisical.AppConnectionAppAzure,
		DestinationConfigAttributes: map[string]schema.Attribute{
			"vault_base_url": schema.StringAttribute{
				Required:    true,
				Description: "The base URL of your Azure Key Vault",
			},
		},
		ReadDestinationConfigForCreateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			destinationConfig := make(map[string]interface{})

			var keyVaultConfig SecretSyncAzureKeyVaultDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &keyVaultConfig, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			destinationConfig["vaultBaseUrl"] = keyVaultConfig.VaultBaseURL.ValueString()

			return destinationConfig, diags
		},
		ReadDestinationConfigForUpdateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel, _ SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			destinationConfig := make(map[string]interface{})

			var keyVaultConfig SecretSyncAzureKeyVaultDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &keyVaultConfig, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			destinationConfig["vaultBaseUrl"] = keyVaultConfig.VaultBaseURL.ValueString()

			return destinationConfig, diags
		},
		ReadDestinationConfigFromApi: func(ctx context.Context, secretSync infisicalclient.SecretSync) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			vaultBaseUrlVal, ok := secretSync.DestinationConfig["vaultBaseUrl"].(string)
			if !ok {
				diags.AddError(
					"Invalid Key Vault Base URL type",
					"Expected 'vaultBaseUrl' to be a string but got something else",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			destinationConfig := map[string]attr.Value{
				"vault_base_url": types.StringValue(vaultBaseUrlVal),
			}

			return types.ObjectValue(map[string]attr.Type{
				"vault_base_url": types.StringType,
			}, destinationConfig)
		},
	}
}
