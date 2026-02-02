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

// AppConnectionAzureDevOpsCredentialsModel describes the data source data model.
type AppConnectionAzureDevOpsCredentialsModel struct {
	OrganizationName types.String `tfsdk:"organization_name"`
	AccessToken      types.String `tfsdk:"access_token"`
	TenantId         types.String `tfsdk:"tenant_id"`
	ClientId         types.String `tfsdk:"client_id"`
	ClientSecret     types.String `tfsdk:"client_secret"`
}

const AzureDevOpsAppConnectionAccessTokenMethod = "access-token"
const AzureDevOpsAppConnectionClientSecretsMethod = "client-secret"

func NewAppConnectionAzureDevOpsResource() resource.Resource {
	return &AppConnectionBaseResource{
		App:               infisical.AppConnectionAppAzureDevOps,
		AppConnectionName: "Azure DevOps",
		ResourceTypeName:  "_app_connection_azure_devops",
		AllowedMethods:    []string{AzureDevOpsAppConnectionAccessTokenMethod, AzureDevOpsAppConnectionClientSecretsMethod},
		CredentialsAttributes: map[string]schema.Attribute{
			"organization_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Azure DevOps organization. For more details, refer to the documentation here infisical.com/docs/integrations/app-connections/azure-devops",
				Sensitive:   false,
			},
			"access_token": schema.StringAttribute{
				Optional:    true,
				Description: "The Azure DevOps access token. Required for access-token method. For more details, refer to the documentation here infisical.com/docs/integrations/app-connections/azure-devops",
				Sensitive:   true,
			},
			"tenant_id": schema.StringAttribute{
				Optional:    true,
				Description: "The Azure Active Directory (AAD) tenant ID. Required for client-secret method. For more details, refer to the documentation here infisical.com/docs/integrations/app-connections/azure-client-secrets",
				Sensitive:   false,
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
		ReadCredentialsForCreateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			credentialsConfig := make(map[string]any)

			var credentials AppConnectionAzureDevOpsCredentialsModel
			diags := plan.Credentials.As(ctx, &credentials, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() == AzureDevOpsAppConnectionClientSecretsMethod {
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
			} else if plan.Method.ValueString() == AzureDevOpsAppConnectionAccessTokenMethod {
				if credentials.AccessToken.IsNull() || credentials.AccessToken.ValueString() == "" {
					diags.AddError(
						"Unable to create Azure app connection",
						"Access token field must be defined in access-token method",
					)
					return nil, diags
				}

				credentialsConfig["accessToken"] = credentials.AccessToken.ValueString()
			}

			if credentials.OrganizationName.IsNull() || credentials.OrganizationName.ValueString() == "" {
				diags.AddError(
					"Unable to create Azure app connection",
					"Organization name field must be defined",
				)
				return nil, diags
			}

			credentialsConfig["orgName"] = credentials.OrganizationName.ValueString()

			return credentialsConfig, diags
		},
		ReadCredentialsForUpdateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel, state AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			credentialsConfig := make(map[string]any)

			var credentialsFromPlan AppConnectionAzureDevOpsCredentialsModel
			diags := plan.Credentials.As(ctx, &credentialsFromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var credentialsFromState AppConnectionAzureDevOpsCredentialsModel
			diags = state.Credentials.As(ctx, &credentialsFromState, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() == AzureDevOpsAppConnectionClientSecretsMethod {
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
						"Tenant ID field must be defined in client-secret method",
					)
					return nil, diags
				}

				if clientId.IsNull() || clientId.ValueString() == "" {
					diags.AddError(
						"Unable to update Azure app connection",
						"Client ID field must be defined in client-secret method",
					)
					return nil, diags
				}

				if clientSecret.IsNull() || clientSecret.ValueString() == "" {
					diags.AddError(
						"Unable to update Azure app connection",
						"Client secret field must be defined in client-secret method",
					)
					return nil, diags
				}

				credentialsConfig["tenantId"] = tenantId.ValueString()
				credentialsConfig["clientId"] = clientId.ValueString()
				credentialsConfig["clientSecret"] = clientSecret.ValueString()
			} else if plan.Method.ValueString() == AzureDevOpsAppConnectionAccessTokenMethod {
				accessToken := credentialsFromPlan.AccessToken
				if credentialsFromPlan.AccessToken.IsUnknown() {
					accessToken = credentialsFromState.AccessToken
				}
				if accessToken.IsNull() || accessToken.ValueString() == "" {
					diags.AddError(
						"Unable to update Azure app connection",
						"Access token field must be defined in access-token method",
					)
					return nil, diags
				}

				credentialsConfig["accessToken"] = accessToken.ValueString()
			}

			organizationName := credentialsFromPlan.OrganizationName
			if credentialsFromPlan.OrganizationName.IsUnknown() {
				organizationName = credentialsFromState.OrganizationName
			}
			if organizationName.IsNull() || organizationName.ValueString() == "" {
				diags.AddError(
					"Unable to update Azure app connection",
					"Organization name field must be defined",
				)
				return nil, diags
			}

			credentialsConfig["orgName"] = organizationName.ValueString()

			return credentialsConfig, diags
		},
		OverwriteCredentialsFields: func(state *AppConnectionBaseResourceModel) diag.Diagnostics {
			credentialsConfig := map[string]attr.Value{
				"organization_name": types.StringNull(),
				"access_token":      types.StringNull(),
				"tenant_id":         types.StringNull(),
				"client_id":         types.StringNull(),
				"client_secret":     types.StringNull(),
			}

			var diags diag.Diagnostics
			state.Credentials, diags = types.ObjectValue(map[string]attr.Type{
				"organization_name": types.StringType,
				"access_token":      types.StringType,
				"tenant_id":         types.StringType,
				"client_id":         types.StringType,
				"client_secret":     types.StringType,
			}, credentialsConfig)

			return diags
		},
	}
}
