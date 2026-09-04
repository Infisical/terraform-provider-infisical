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

type SecretRotationAuth0ClientSecretParametersModel struct {
	ClientId types.String `tfsdk:"client_id"`
}

type SecretRotationAuth0ClientSecretSecretsMappingModel struct {
	ClientId     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
}

func NewSecretRotationAuth0ClientSecretResource() resource.Resource {
	return &SecretRotationBaseResource{
		Provider:           infisical.SecretRotationProviderAuth0ClientSecret,
		SecretRotationName: "Auth0 Client Secret",
		ResourceTypeName:   "_secret_rotation_auth0_client_secret",
		AppConnection:      infisical.AppConnectionAppAuth0,
		ParametersAttributes: map[string]schema.Attribute{
			"client_id": schema.StringAttribute{
				Required:    true,
				Description: "The client ID of the Auth0 application to rotate the client secret for.",
			},
		},
		SecretsMappingAttributes: map[string]schema.Attribute{
			"client_id": schema.StringAttribute{
				Required:    true,
				Description: "The name of the secret that the client ID will be mapped to.",
			},
			"client_secret": schema.StringAttribute{
				Required:    true,
				Description: "The name of the secret that the rotated client secret will be mapped to.",
			},
		},

		ReadParametersFromPlan: func(ctx context.Context, plan SecretRotationBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			parametersMap := make(map[string]interface{})
			var parameters SecretRotationAuth0ClientSecretParametersModel

			diags := plan.Parameters.As(ctx, &parameters, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			parametersMap["clientId"] = parameters.ClientId.ValueString()

			return parametersMap, diags
		},

		ReadParametersFromApi: func(ctx context.Context, secretRotation infisical.SecretRotation) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics
			parameters := make(map[string]attr.Value)
			parametersSchema := map[string]attr.Type{
				"client_id": types.StringType,
			}

			clientIdVal, ok := secretRotation.Parameters["clientId"].(string)
			if !ok {
				diags.AddError("API Reading Error", "Expected 'clientId' (string) but got wrong type or missing")
				return types.ObjectNull(parametersSchema), diags
			}
			parameters["client_id"] = types.StringValue(clientIdVal)

			obj, objDiags := types.ObjectValue(parametersSchema, parameters)
			diags.Append(objDiags...)
			if diags.HasError() {
				return types.ObjectNull(parametersSchema), diags
			}

			return obj, diags
		},

		ReadSecretsMappingFromPlan: func(ctx context.Context, plan SecretRotationBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			secretsMappingMap := make(map[string]interface{})
			var secretsMapping SecretRotationAuth0ClientSecretSecretsMappingModel

			diags := plan.SecretsMapping.As(ctx, &secretsMapping, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			secretsMappingMap["clientId"] = secretsMapping.ClientId.ValueString()
			secretsMappingMap["clientSecret"] = secretsMapping.ClientSecret.ValueString()

			return secretsMappingMap, diags
		},

		ReadSecretsMappingFromApi: func(ctx context.Context, secretRotation infisical.SecretRotation) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics
			secretsMapping := make(map[string]attr.Value)
			secretsMappingSchema := map[string]attr.Type{
				"client_id":     types.StringType,
				"client_secret": types.StringType,
			}

			clientIdVal, ok := secretRotation.SecretsMapping["clientId"].(string)
			if !ok {
				diags.AddError("API Reading Error", "Expected 'clientId' (string) but got wrong type or missing")
				return types.ObjectNull(secretsMappingSchema), diags
			}
			secretsMapping["client_id"] = types.StringValue(clientIdVal)

			clientSecretVal, ok := secretRotation.SecretsMapping["clientSecret"].(string)
			if !ok {
				diags.AddError("API Reading Error", "Expected 'clientSecret' (string) but got wrong type or missing")
				return types.ObjectNull(secretsMappingSchema), diags
			}
			secretsMapping["client_secret"] = types.StringValue(clientSecretVal)

			obj, objDiags := types.ObjectValue(secretsMappingSchema, secretsMapping)
			diags.Append(objDiags...)
			if diags.HasError() {
				return types.ObjectNull(secretsMappingSchema), diags
			}

			return obj, diags
		},
	}
}
