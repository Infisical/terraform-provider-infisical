package resource

import (
	"context"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// AppConnectionAzureKeyVaultCredentialsModel describes the data source data model.
type AppConnectionAzureKeyVaultCredentialsModel struct {
	TenantId     types.String `tfsdk:"tenant_id"`
	ClientId     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
}

const AzureKeyVaultAppConnectionClientSecretsMethod = "client-secret"

func NewAppConnectionAzureKeyVaultResource() resource.Resource {
	return &AppConnectionBaseResource{
		App:               infisical.AppConnectionAppAzureKeyVault,
		AppConnectionName: "Azure Key Vault",
		ResourceTypeName:  "_app_connection_azure_key_vault",
		AllowedMethods:    []string{AzureKeyVaultAppConnectionClientSecretsMethod},
		CredentialsAttributes: map[string]schema.Attribute{
			"tenant_id": schema.StringAttribute{
				Required:    true,
				Description: "The Azure Active Directory (AAD) tenant ID. Required for key-vault method. For more details, refer to the documentation here infisical.com/docs/integrations/app-connections/azure-key-vault",
				Sensitive:   false,
			},
			"client_id": schema.StringAttribute{
				Required:    true,
				Description: "The Azure application (client) ID. Required for key-vault method. For more details, refer to the documentation here infisical.com/docs/integrations/app-connections/azure-key-vault",
				Sensitive:   true,
			},
			"client_secret": schema.StringAttribute{
				Required:    true,
				Description: "The Azure client secret. Required for key-vault method. For more details, refer to the documentation here infisical.com/docs/integrations/app-connections/azure-key-vault",
				Sensitive:   true,
			},
		},
		ReadCredentialsForCreateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			credentialsConfig := make(map[string]any)

			var credentials AppConnectionAzureKeyVaultCredentialsModel
			diags := plan.Credentials.As(ctx, &credentials, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() == AzureKeyVaultAppConnectionClientSecretsMethod {
				if credentials.TenantId.IsNull() || credentials.TenantId.ValueString() == "" {
					diags.AddError(
						"Unable to create Azure app connection",
						"Tenant ID field must be defined in key-vault method",
					)
					return nil, diags
				}

				if credentials.ClientId.IsNull() || credentials.ClientId.ValueString() == "" {
					diags.AddError(
						"Unable to create Azure app connection",
						"Client ID field must be defined in key-vault method",
					)
					return nil, diags
				}

				if credentials.ClientSecret.IsNull() || credentials.ClientSecret.ValueString() == "" {
					diags.AddError(
						"Unable to create Azure app connection",
						"Client secret field must be defined in key-vault method",
					)
					return nil, diags
				}

				credentialsConfig["tenantId"] = credentials.TenantId.ValueString()
				credentialsConfig["clientId"] = credentials.ClientId.ValueString()
				credentialsConfig["clientSecret"] = credentials.ClientSecret.ValueString()
			}

			return credentialsConfig, diags
		},
		ReadCredentialsForUpdateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel, state AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			credentialsConfig := make(map[string]any)

			var credentialsFromPlan AppConnectionAzureKeyVaultCredentialsModel
			diags := plan.Credentials.As(ctx, &credentialsFromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var credentialsFromState AppConnectionAzureKeyVaultCredentialsModel
			diags = state.Credentials.As(ctx, &credentialsFromState, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() == AzureKeyVaultAppConnectionClientSecretsMethod {
				tenantId := credentialsFromPlan.TenantId
				if credentialsFromPlan.TenantId.IsUnknown() {
					tenantId = credentialsFromState.TenantId
				}

				clientId := credentialsFromPlan.ClientId
				if credentialsFromPlan.ClientId.IsUnknown() {
					clientId = credentialsFromState.ClientId
				}

				clientSecret := credentialsFromPlan.ClientSecret
				if credentialsFromPlan.ClientSecret.IsUnknown() {
					clientSecret = credentialsFromState.ClientSecret
				}

				if tenantId.IsNull() || tenantId.ValueString() == "" {
					diags.AddError(
						"Unable to update Azure app connection",
						"Tenant ID field must be defined in key-vault method",
					)
					return nil, diags
				}

				if clientId.IsNull() || clientId.ValueString() == "" {
					diags.AddError(
						"Unable to update Azure app connection",
						"Client ID field must be defined in key-vault method",
					)
					return nil, diags
				}

				if clientSecret.IsNull() || clientSecret.ValueString() == "" {
					diags.AddError(
						"Unable to update Azure app connection",
						"Client secret field must be defined in key-vault method",
					)
					return nil, diags
				}

				credentialsConfig["tenantId"] = tenantId.ValueString()
				credentialsConfig["clientId"] = clientId.ValueString()
				credentialsConfig["clientSecret"] = clientSecret.ValueString()
			}

			return credentialsConfig, diags
		},
		OverwriteCredentialsFields: func(state *AppConnectionBaseResourceModel) diag.Diagnostics {
			credentialsConfig := map[string]attr.Value{
				"tenant_id":     types.StringNull(),
				"client_id":     types.StringNull(),
				"client_secret": types.StringNull(),
			}

			var diags diag.Diagnostics
			state.Credentials, diags = types.ObjectValue(map[string]attr.Type{
				"tenant_id":     types.StringType,
				"client_id":     types.StringType,
				"client_secret": types.StringType,
			}, credentialsConfig)

			return diags
		},
	}
}
