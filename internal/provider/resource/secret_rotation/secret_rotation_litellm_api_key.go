package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type SecretRotationLiteLLMApiKeyParametersModel struct {
	Name              types.String `tfsdk:"name"`
	UserID            types.String `tfsdk:"user_id"`
	TeamID            types.String `tfsdk:"team_id"`
	Models            types.List   `tfsdk:"models"`
	AdditionalOptions types.String `tfsdk:"additional_options"`
}

type SecretRotationLiteLLMApiKeySecretsMappingModel struct {
	ApiKey types.String `tfsdk:"api_key"`
}

func NewSecretRotationLiteLLMApiKeyResource() resource.Resource {
	return &SecretRotationBaseResource{
		Provider:           infisical.SecretRotationProviderLiteLLMApiKey,
		SecretRotationName: "LiteLLM API Key",
		ResourceTypeName:   "_secret_rotation_litellm_api_key",
		AppConnection:      infisical.AppConnectionAppLiteLLM,
		ParametersAttributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name for the generated LiteLLM API key. Infisical appends a timestamp to keep each rotated key unique and record its creation time.",
			},
			"user_id": schema.StringAttribute{
				Optional:    true,
				Description: "The ID of the LiteLLM user to associate the generated key with.",
			},
			"team_id": schema.StringAttribute{
				Optional:    true,
				Description: "The ID of the LiteLLM team to associate the generated key with.",
			},
			"models": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "The list of model names the generated key is allowed to access. An empty or omitted list allows all models.",
			},
			"additional_options": schema.StringAttribute{
				Optional:    true,
				Description: "A JSON-encoded object of additional LiteLLM /key/generate options (e.g. max_budget, tpm_limit, metadata). Use jsonencode() to build it. Reserved fields (key_alias, auto_rotate, rotation_interval, duration, send_invite_email, key_type, user_id, team_id, models) and permission fields (allowed_routes, permission, object_permission) cannot be set here.",
			},
		},
		SecretsMappingAttributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Required:    true,
				Description: "The name of the secret that the rotated LiteLLM API key will be mapped to.",
			},
		},

		ReadParametersFromPlan: func(ctx context.Context, plan SecretRotationBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			parametersMap := make(map[string]interface{})
			var parameters SecretRotationLiteLLMApiKeyParametersModel

			diags := plan.Parameters.As(ctx, &parameters, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			parametersMap["name"] = parameters.Name.ValueString()

			if !parameters.UserID.IsNull() && !parameters.UserID.IsUnknown() {
				parametersMap["userId"] = parameters.UserID.ValueString()
			}

			if !parameters.TeamID.IsNull() && !parameters.TeamID.IsUnknown() {
				parametersMap["teamId"] = parameters.TeamID.ValueString()
			}

			if !parameters.Models.IsNull() && !parameters.Models.IsUnknown() {
				models := make([]string, 0, len(parameters.Models.Elements()))
				diags.Append(parameters.Models.ElementsAs(ctx, &models, false)...)
				if diags.HasError() {
					return nil, diags
				}
				parametersMap["models"] = models
			}

			if !parameters.AdditionalOptions.IsNull() && !parameters.AdditionalOptions.IsUnknown() {
				parametersMap["additionalOptions"] = parameters.AdditionalOptions.ValueString()
			}

			return parametersMap, diags
		},

		ReadParametersFromApi: func(ctx context.Context, secretRotation infisical.SecretRotation) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics
			parameters := make(map[string]attr.Value)
			parametersSchema := map[string]attr.Type{
				"name":               types.StringType,
				"user_id":            types.StringType,
				"team_id":            types.StringType,
				"models":             types.ListType{ElemType: types.StringType},
				"additional_options": types.StringType,
			}

			nameVal, ok := secretRotation.Parameters["name"].(string)
			if !ok {
				diags.AddError("API Reading Error", "Expected 'name' (string) but got wrong type or missing")
				return types.ObjectNull(parametersSchema), diags
			}
			parameters["name"] = types.StringValue(nameVal)

			optionalStrings := map[string]string{
				"userId":            "user_id",
				"teamId":            "team_id",
				"additionalOptions": "additional_options",
			}
			for apiKey, tfKey := range optionalStrings {
				raw, exists := secretRotation.Parameters[apiKey]
				if !exists || raw == nil {
					parameters[tfKey] = types.StringNull()
					continue
				}
				val, ok := raw.(string)
				if !ok {
					diags.AddError("API Reading Error", fmt.Sprintf("Expected '%s' (string) but got wrong type", apiKey))
					return types.ObjectNull(parametersSchema), diags
				}
				parameters[tfKey] = types.StringValue(val)
			}

			rawModels, exists := secretRotation.Parameters["models"]
			if !exists || rawModels == nil {
				parameters["models"] = types.ListNull(types.StringType)
			} else {
				modelsSlice, ok := rawModels.([]interface{})
				if !ok {
					diags.AddError("API Reading Error", "Expected 'models' (array) but got wrong type")
					return types.ObjectNull(parametersSchema), diags
				}
				modelValues := make([]attr.Value, 0, len(modelsSlice))
				for _, m := range modelsSlice {
					modelStr, ok := m.(string)
					if !ok {
						diags.AddError("API Reading Error", "Expected 'models' to contain only strings")
						return types.ObjectNull(parametersSchema), diags
					}
					modelValues = append(modelValues, types.StringValue(modelStr))
				}
				modelsList, listDiags := types.ListValue(types.StringType, modelValues)
				diags.Append(listDiags...)
				if diags.HasError() {
					return types.ObjectNull(parametersSchema), diags
				}
				parameters["models"] = modelsList
			}

			obj, objDiags := types.ObjectValue(parametersSchema, parameters)
			diags.Append(objDiags...)
			if diags.HasError() {
				return types.ObjectNull(parametersSchema), diags
			}

			return obj, diags
		},

		ReadSecretsMappingFromPlan: func(ctx context.Context, plan SecretRotationBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			secretsMappingMap := make(map[string]interface{})
			var secretsMapping SecretRotationLiteLLMApiKeySecretsMappingModel

			diags := plan.SecretsMapping.As(ctx, &secretsMapping, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			secretsMappingMap["apiKey"] = secretsMapping.ApiKey.ValueString()

			return secretsMappingMap, diags
		},

		ReadSecretsMappingFromApi: func(ctx context.Context, secretRotation infisical.SecretRotation) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics
			secretsMapping := make(map[string]attr.Value)
			secretsMappingSchema := map[string]attr.Type{
				"api_key": types.StringType,
			}

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
