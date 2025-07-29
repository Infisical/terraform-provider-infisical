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

// AppConnectionAzureCredentialsModel describes the data source data model.
type AppConnectionAzureCredentialsModel struct {
	TenantId     types.String `tfsdk:"tenant_id"`
	ClientId     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
}

const AzureAppConnectionClientSecretsMethod = "client-secret"

func NewAppConnectionAzureResource() resource.Resource {
	return &AppConnectionBaseResource{
		App:               infisical.AppConnectionAppAzureClientSecrets,
		AppConnectionName: "Azure Client Secrets",
		ResourceTypeName:  "_app_connection_azure_client_secrets",
		AllowedMethods:    []string{AzureAppConnectionClientSecretsMethod},
		CredentialsAttributes: map[string]schema.Attribute{
			"tenant_id": schema.StringAttribute{
				Optional:    true,
				Description: "The Azure Active Directory (AAD) tenant ID. Required for client-secret method. For more details, refer to the documentation here infisical.com/docs/integrations/app-connections/azure-client-secrets",
				Sensitive:   true,
			},
			"client_id": schema.StringAttribute{
				Optional:    true,
				Description: "The Azure application (client) ID. Required for client-secret method. For more details, refer to the documentation here infisical.com/docs/integrations/app-connections/azure-client-secrets",
				Sensitive:   true,
			},
			"client_secret": schema.StringAttribute{
				Optional:    true,
				Description: "The Azure client secret. Required for client-secret method. For more details, refer to the documentation here infisical.com/docs/integrations/app-connections/azure-client-secrets",
				Sensitive:   true,
			},
		},
		ReadCredentialsForCreateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			credentialsConfig := make(map[string]interface{})

			var credentials AppConnectionAzureCredentialsModel
			diags := plan.Credentials.As(ctx, &credentials, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() == AzureAppConnectionClientSecretsMethod {
				if credentials.TenantId.IsNull() || credentials.TenantId.ValueString() == "" {
					diags.AddError(
						"Unable to create Azure app connection",
						"Tenant ID field must be defined in client-secret method",
					)
					return nil, diags
				}

				if credentials.ClientId.IsNull() || credentials.ClientId.ValueString() == "" {
					diags.AddError(
						"Unable to create Azure app connection",
						"Client ID field must be defined in client-secret method",
					)
					return nil, diags
				}

				if credentials.ClientSecret.IsNull() || credentials.ClientSecret.ValueString() == "" {
					diags.AddError(
						"Unable to create Azure app connection",
						"Client secret field must be defined in client-secret method",
					)
					return nil, diags
				}

				credentialsConfig["tenantId"] = credentials.TenantId.ValueString()
				credentialsConfig["clientId"] = credentials.ClientId.ValueString()
				credentialsConfig["clientSecret"] = credentials.ClientSecret.ValueString()
			}

			return credentialsConfig, diags
		},
		ReadCredentialsForUpdateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel, state AppConnectionBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			credentialsConfig := make(map[string]interface{})

			var credentialsFromPlan AppConnectionAzureCredentialsModel
			diags := plan.Credentials.As(ctx, &credentialsFromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var credentialsFromState AppConnectionAzureCredentialsModel
			diags = state.Credentials.As(ctx, &credentialsFromState, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() == AzureAppConnectionClientSecretsMethod {
				if credentialsFromPlan.TenantId.IsNull() || credentialsFromPlan.TenantId.ValueString() == "" {
					diags.AddError(
						"Unable to update Azure app connection",
						"Tenant ID field must be defined in client-secret method",
					)
					return nil, diags
				}

				if credentialsFromPlan.ClientId.IsNull() || credentialsFromPlan.ClientId.ValueString() == "" {
					diags.AddError(
						"Unable to update Azure app connection",
						"Client ID field must be defined in client-secret method",
					)
					return nil, diags
				}

				if credentialsFromPlan.ClientSecret.IsNull() || credentialsFromPlan.ClientSecret.ValueString() == "" {
					diags.AddError(
						"Unable to update Azure app connection",
						"Client secret field must be defined in client-secret method",
					)
					return nil, diags
				}

				credentialsConfig["tenantId"] = credentialsFromPlan.TenantId.ValueString()
				credentialsConfig["clientId"] = credentialsFromPlan.ClientId.ValueString()
				credentialsConfig["clientSecret"] = credentialsFromPlan.ClientSecret.ValueString()
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
