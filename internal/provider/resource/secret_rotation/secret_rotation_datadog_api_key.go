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

type SecretRotationDatadogApiKeyParametersModel struct {
	Name types.String `tfsdk:"name"`
}

type SecretRotationDatadogApiKeySecretsMappingModel struct {
	ApiKeyID types.String `tfsdk:"api_key_id"`
	ApiKey   types.String `tfsdk:"api_key"`
}

func NewSecretRotationDatadogApiKeyResource() resource.Resource {
	return &SecretRotationBaseResource{
		Provider:           infisical.SecretRotationProviderDatadogApiKey,
		SecretRotationName: "Datadog API Key",
		ResourceTypeName:   "_secret_rotation_datadog_api_key",
		AppConnection:      infisical.AppConnectionAppDatadog,
		ParametersAttributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name for the generated Datadog API keys.",
			},
		},
		SecretsMappingAttributes: map[string]schema.Attribute{
			"api_key_id": schema.StringAttribute{
				Required:    true,
				Description: "The name of the secret that the Datadog API key ID will be mapped to.",
			},
			"api_key": schema.StringAttribute{
				Required:    true,
				Description: "The name of the secret that the rotated Datadog API key will be mapped to.",
			},
		},

		ReadParametersFromPlan: func(ctx context.Context, plan SecretRotationBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			parametersMap := make(map[string]interface{})
			var parameters SecretRotationDatadogApiKeyParametersModel

			diags := plan.Parameters.As(ctx, &parameters, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			parametersMap["name"] = parameters.Name.ValueString()

			return parametersMap, diags
		},

		ReadParametersFromApi: func(ctx context.Context, secretRotation infisical.SecretRotation) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics
			parameters := make(map[string]attr.Value)
			parametersSchema := map[string]attr.Type{
				"name": types.StringType,
			}

			nameVal, ok := secretRotation.Parameters["name"].(string)
			if !ok {
				diags.AddError("API Reading Error", "Expected 'name' (string) but got wrong type or missing")
				return types.ObjectNull(parametersSchema), diags
			}
			parameters["name"] = types.StringValue(nameVal)

			obj, objDiags := types.ObjectValue(parametersSchema, parameters)
			diags.Append(objDiags...)
			if diags.HasError() {
				return types.ObjectNull(parametersSchema), diags
			}

			return obj, diags
		},

		ReadSecretsMappingFromPlan: func(ctx context.Context, plan SecretRotationBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			secretsMappingMap := make(map[string]interface{})
			var secretsMapping SecretRotationDatadogApiKeySecretsMappingModel

			diags := plan.SecretsMapping.As(ctx, &secretsMapping, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			secretsMappingMap["apiKeyId"] = secretsMapping.ApiKeyID.ValueString()
			secretsMappingMap["apiKey"] = secretsMapping.ApiKey.ValueString()

			return secretsMappingMap, diags
		},

		ReadSecretsMappingFromApi: func(ctx context.Context, secretRotation infisical.SecretRotation) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics
			secretsMapping := make(map[string]attr.Value)
			secretsMappingSchema := map[string]attr.Type{
				"api_key_id": types.StringType,
				"api_key":    types.StringType,
			}

			apiKeyIdVal, ok := secretRotation.SecretsMapping["apiKeyId"].(string)
			if !ok {
				diags.AddError("API Reading Error", "Expected 'apiKeyId' (string) but got wrong type or missing")
				return types.ObjectNull(secretsMappingSchema), diags
			}
			secretsMapping["api_key_id"] = types.StringValue(apiKeyIdVal)

			apiKeyVal, ok := secretRotation.SecretsMapping["apiKey"].(string)
			if !ok {
				diags.AddError("API Reading Error", "Expected 'apiKey' (string) but got wrong type or missing")
				return types.ObjectNull(secretsMappingSchema), diags
			}
			secretsMapping["api_key"] = types.StringValue(apiKeyVal)

			obj, objDiags := types.ObjectValue(secretsMappingSchema, secretsMapping)
			diags.Append(objDiags...)
			if diags.HasError() {
				return types.ObjectNull(secretsMappingSchema), diags
			}

			return obj, diags
		},
	}
}
