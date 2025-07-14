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

type SecretRotationAzureClientSecretParametersModel struct {
	ObjectId types.String `tfsdk:"object_id"`
	ClientId types.String `tfsdk:"client_id"`
}

type SecretRotationAzureClientSecretSecretsMappingModel struct {
	ClientId     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
}

func NewSecretRotationAzureClientSecretResource() resource.Resource {
	return &SecretRotationBaseResource{
		Provider:           infisical.SecretRotationProviderAzureClientSecret,
		SecretRotationName: "Azure Client Secret",
		ResourceTypeName:   "_secret_rotation_azure_client_secret",
		AppConnection:      infisical.AppConnectionAppAzureClientSecrets,
		ParametersAttributes: map[string]schema.Attribute{
			"object_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the Azure Application to rotate the client secret for.",
			},
			"client_id": schema.StringAttribute{
				Required:    true,
				Description: "The client ID of the Azure Application to rotate the client secret for.",
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
			var parameters SecretRotationAzureClientSecretParametersModel

			diags := plan.Parameters.As(ctx, &parameters, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			parametersMap["objectId"] = parameters.ObjectId.ValueString()
			parametersMap["clientId"] = parameters.ClientId.ValueString()

			return parametersMap, diags
		},

		ReadParametersFromApi: func(ctx context.Context, secretRotation infisicalclient.SecretRotation) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics
			parameters := make(map[string]attr.Value)
			parametersSchema := map[string]attr.Type{
				"object_id": types.StringType,
				"client_id": types.StringType,
			}

			objectIdVal, ok := secretRotation.Parameters["objectId"].(string)
			if !ok {
				diags.AddError("API Reading Error", "Expected 'objectId' (string) but got wrong type or missing")
				return types.ObjectNull(parametersSchema), diags
			}
			parameters["object_id"] = types.StringValue(objectIdVal)

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
			var secretsMapping SecretRotationAzureClientSecretSecretsMappingModel

			diags := plan.SecretsMapping.As(ctx, &secretsMapping, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			secretsMappingMap["clientId"] = secretsMapping.ClientId.ValueString()
			secretsMappingMap["clientSecret"] = secretsMapping.ClientSecret.ValueString()

			return secretsMappingMap, diags
		},

		ReadSecretsMappingFromApi: func(ctx context.Context, secretRotation infisicalclient.SecretRotation) (types.Object, diag.Diagnostics) {
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
