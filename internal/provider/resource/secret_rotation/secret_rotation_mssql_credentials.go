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

type SecretRotationMsSqlCredentialsParametersModel struct {
	Username1 types.String `tfsdk:"username1"`
	Username2 types.String `tfsdk:"username2"`
}

type SecretRotationMsSqlCredentialsSecretsMappingModel struct {
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

func NewSecretRotationMsSqlCredentialsResource() resource.Resource {
	return &SecretRotationBaseResource{
		Provider:           infisical.SecretRotationProviderMsSqlCredentials,
		SecretRotationName: "MsSQL Credentials",
		ResourceTypeName:   "_secret_rotation_mssql_credentials",
		AppConnection:      infisical.AppConnectionAppMsSql,
		ParametersAttributes: map[string]schema.Attribute{
			"username1": schema.StringAttribute{
				Required:    true,
				Description: "The username of the first login to rotate passwords for. This user must already exists in your database.",
			},
			"username2": schema.StringAttribute{
				Required:    true,
				Description: "The username of the second login to rotate passwords for. This user must already exists in your database.",
			},
		},
		SecretsMappingAttributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				Required:    true,
				Description: "The name of the secret that the active username will be mapped to.",
			},
			"password": schema.StringAttribute{
				Required:    true,
				Description: "The name of the secret that the generated password will be mapped to.",
			},
		},

		ReadParametersFromPlan: func(ctx context.Context, plan SecretRotationBaseResourceModel) (map[string]any, diag.Diagnostics) {
			parametersMap := make(map[string]any)
			var parameters SecretRotationMsSqlCredentialsParametersModel

			diags := plan.Parameters.As(ctx, &parameters, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			parametersMap["username1"] = parameters.Username1.ValueString()
			parametersMap["username2"] = parameters.Username2.ValueString()

			return parametersMap, diags
		},

		ReadParametersFromApi: func(ctx context.Context, secretRotation infisicalclient.SecretRotation) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics
			parameters := make(map[string]attr.Value)
			parametersSchema := map[string]attr.Type{
				"username1": types.StringType,
				"username2": types.StringType,
			}

			usernameOneVal, ok := secretRotation.Parameters["username1"].(string)
			if !ok {
				diags.AddError("API Reading Error", "Expected 'username1' (string) but got wrong type or missing")
				return types.ObjectNull(parametersSchema), diags
			}
			parameters["username1"] = types.StringValue(usernameOneVal)

			usernameTwoVal, ok := secretRotation.Parameters["username2"].(string)
			if !ok {
				diags.AddError("API Reading Error", "Expected 'username2' (string) but got wrong type or missing")
				return types.ObjectNull(parametersSchema), diags
			}
			parameters["username2"] = types.StringValue(usernameTwoVal)

			obj, objDiags := types.ObjectValue(parametersSchema, parameters)
			diags.Append(objDiags...)
			if diags.HasError() {
				return types.ObjectNull(parametersSchema), diags
			}

			return obj, diags
		},

		ReadSecretsMappingFromPlan: func(ctx context.Context, plan SecretRotationBaseResourceModel) (map[string]any, diag.Diagnostics) {
			secretsMappingMap := make(map[string]any)
			var secretsMapping SecretRotationMsSqlCredentialsSecretsMappingModel

			diags := plan.SecretsMapping.As(ctx, &secretsMapping, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			secretsMappingMap["username"] = secretsMapping.Username.ValueString()
			secretsMappingMap["password"] = secretsMapping.Password.ValueString()

			return secretsMappingMap, diags
		},

		ReadSecretsMappingFromApi: func(ctx context.Context, secretRotation infisicalclient.SecretRotation) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics
			secretsMapping := make(map[string]attr.Value)
			secretsMappingSchema := map[string]attr.Type{
				"username": types.StringType,
				"password": types.StringType,
			}

			usernameVal, ok := secretRotation.SecretsMapping["username"].(string)
			if !ok {
				diags.AddError("API Reading Error", "Expected 'username' (string) but got wrong type or missing")
				return types.ObjectNull(secretsMappingSchema), diags
			}
			secretsMapping["username"] = types.StringValue(usernameVal)

			passwordVal, ok := secretRotation.SecretsMapping["password"].(string)
			if !ok {
				diags.AddError("API Reading Error", "Expected 'password' (string) but got wrong type or missing")
				return types.ObjectNull(secretsMappingSchema), diags
			}
			secretsMapping["password"] = types.StringValue(passwordVal)

			obj, objDiags := types.ObjectValue(secretsMappingSchema, secretsMapping)
			diags.Append(objDiags...)
			if diags.HasError() {
				return types.ObjectNull(secretsMappingSchema), diags
			}

			return obj, diags
		},
	}
}
