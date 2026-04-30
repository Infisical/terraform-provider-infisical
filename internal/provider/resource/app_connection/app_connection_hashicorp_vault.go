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

type AppConnectionHashicorpVaultCredentialsModel struct {
	InstanceUrl types.String `tfsdk:"instance_url"`
	Namespace   types.String `tfsdk:"namespace"`
	AccessToken types.String `tfsdk:"access_token"`
	RoleId      types.String `tfsdk:"role_id"`
	SecretId    types.String `tfsdk:"secret_id"`
}

const HashicorpVaultAppConnectionAccessTokenMethod = "access-token"
const HashicorpVaultAppConnectionAppRoleMethod = "app-role"

func NewAppConnectionHashicorpVaultResource() resource.Resource {
	return &AppConnectionBaseResource{
		App:               infisical.AppConnectionAppHashicorpVault,
		AppConnectionName: "HashiCorp Vault",
		ResourceTypeName:  "_app_connection_hashicorp_vault",
		AllowedMethods:    []string{HashicorpVaultAppConnectionAccessTokenMethod, HashicorpVaultAppConnectionAppRoleMethod},
		CredentialsAttributes: map[string]schema.Attribute{
			"instance_url": schema.StringAttribute{
				Optional:    true,
				Description: "The URL of the HashiCorp Vault instance, e.g. `https://vault.example.com`. Required for all methods.",
			},
			"namespace": schema.StringAttribute{
				Optional:    true,
				Description: "Optional Vault namespace. Only applicable to HCP Vault Dedicated and Enterprise deployments.",
			},
			"access_token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "The Vault access token. Required for the `access-token` method.",
			},
			"role_id": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "The AppRole role ID. Required for the `app-role` method.",
			},
			"secret_id": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "The AppRole secret ID. Required for the `app-role` method.",
			},
		},
		ReadCredentialsForCreateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			credentialsConfig := make(map[string]any)

			var credentials AppConnectionHashicorpVaultCredentialsModel
			diags := plan.Credentials.As(ctx, &credentials, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if credentials.InstanceUrl.IsNull() || credentials.InstanceUrl.ValueString() == "" {
				diags.AddError(
					"Unable to create HashiCorp Vault app connection",
					"Instance url field must be defined for all methods",
				)
				return nil, diags
			}

			credentialsConfig["instanceUrl"] = credentials.InstanceUrl.ValueString()

			if !credentials.Namespace.IsNull() && credentials.Namespace.ValueString() != "" {
				credentialsConfig["namespace"] = credentials.Namespace.ValueString()
			}

			if plan.Method.ValueString() == HashicorpVaultAppConnectionAccessTokenMethod {
				if credentials.AccessToken.IsNull() || credentials.AccessToken.ValueString() == "" {
					diags.AddError(
						"Unable to create HashiCorp Vault app connection",
						"Access token field must be defined in access-token method",
					)
					return nil, diags
				}

				credentialsConfig["accessToken"] = credentials.AccessToken.ValueString()
			} else {
				if credentials.RoleId.IsNull() || credentials.RoleId.ValueString() == "" {
					diags.AddError(
						"Unable to create HashiCorp Vault app connection",
						"Role id field must be defined in app-role method",
					)
					return nil, diags
				}

				if credentials.SecretId.IsNull() || credentials.SecretId.ValueString() == "" {
					diags.AddError(
						"Unable to create HashiCorp Vault app connection",
						"Secret id field must be defined in app-role method",
					)
					return nil, diags
				}

				credentialsConfig["roleId"] = credentials.RoleId.ValueString()
				credentialsConfig["secretId"] = credentials.SecretId.ValueString()
			}

			return credentialsConfig, diags
		},
		ReadCredentialsForUpdateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel, state AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			credentialsConfig := make(map[string]any)

			var credentialsFromPlan AppConnectionHashicorpVaultCredentialsModel
			diags := plan.Credentials.As(ctx, &credentialsFromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var credentialsFromState AppConnectionHashicorpVaultCredentialsModel
			diags = state.Credentials.As(ctx, &credentialsFromState, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			instanceUrl := credentialsFromPlan.InstanceUrl
			if credentialsFromPlan.InstanceUrl.IsUnknown() {
				instanceUrl = credentialsFromState.InstanceUrl
			}

			if instanceUrl.IsNull() || instanceUrl.ValueString() == "" {
				diags.AddError(
					"Unable to update HashiCorp Vault app connection",
					"Instance url field must be defined for all methods",
				)
				return nil, diags
			}

			credentialsConfig["instanceUrl"] = instanceUrl.ValueString()

			namespace := credentialsFromPlan.Namespace
			if credentialsFromPlan.Namespace.IsUnknown() {
				namespace = credentialsFromState.Namespace
			}
			if !namespace.IsNull() && namespace.ValueString() != "" {
				credentialsConfig["namespace"] = namespace.ValueString()
			}

			if plan.Method.ValueString() == HashicorpVaultAppConnectionAccessTokenMethod {
				accessToken := credentialsFromPlan.AccessToken
				if credentialsFromPlan.AccessToken.IsUnknown() {
					accessToken = credentialsFromState.AccessToken
				}

				if accessToken.IsNull() || accessToken.ValueString() == "" {
					diags.AddError(
						"Unable to update HashiCorp Vault app connection",
						"Access token field must be defined in access-token method",
					)
					return nil, diags
				}

				credentialsConfig["accessToken"] = accessToken.ValueString()
			} else {
				roleId := credentialsFromPlan.RoleId
				if credentialsFromPlan.RoleId.IsUnknown() {
					roleId = credentialsFromState.RoleId
				}

				secretId := credentialsFromPlan.SecretId
				if credentialsFromPlan.SecretId.IsUnknown() {
					secretId = credentialsFromState.SecretId
				}

				if roleId.IsNull() || roleId.ValueString() == "" {
					diags.AddError(
						"Unable to update HashiCorp Vault app connection",
						"Role id field must be defined in app-role method",
					)
					return nil, diags
				}

				if secretId.IsNull() || secretId.ValueString() == "" {
					diags.AddError(
						"Unable to update HashiCorp Vault app connection",
						"Secret id field must be defined in app-role method",
					)
					return nil, diags
				}

				credentialsConfig["roleId"] = roleId.ValueString()
				credentialsConfig["secretId"] = secretId.ValueString()
			}

			return credentialsConfig, diags
		},
		OverwriteCredentialsFields: func(state *AppConnectionBaseResourceModel) diag.Diagnostics {
			credentialsConfig := map[string]attr.Value{
				"instance_url": types.StringNull(),
				"namespace":    types.StringNull(),
				"access_token": types.StringNull(),
				"role_id":      types.StringNull(),
				"secret_id":    types.StringNull(),
			}

			var diags diag.Diagnostics
			state.Credentials, diags = types.ObjectValue(map[string]attr.Type{
				"instance_url": types.StringType,
				"namespace":    types.StringType,
				"access_token": types.StringType,
				"role_id":      types.StringType,
				"secret_id":    types.StringType,
			}, credentialsConfig)

			return diags
		},
	}
}
