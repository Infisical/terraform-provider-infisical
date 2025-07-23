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

type SecretRotationAwsIAMSecretParametersModel struct {
	UserName types.String `tfsdk:"user_name"`
	Region   types.String `tfsdk:"region"`
}

type SecretRotationAwsIAMSecretSecretsMappingModel struct {
	AccessKeyID     types.String `tfsdk:"access_key_id"`
	SecretAccessKey types.String `tfsdk:"secret_access_key"`
}

func NewSecretRotationAwsIamUserSecretResource() resource.Resource {
	return &SecretRotationBaseResource{
		Provider:           infisical.SecretRotationProviderAwsIamUserSecret,
		SecretRotationName: "AWS IAM User Secret",
		ResourceTypeName:   "_secret_rotation_aws_iam_user_secret",
		AppConnection:      infisical.AppConnectionAppAWS,
		ParametersAttributes: map[string]schema.Attribute{
			"user_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the client to rotate credentials for.",
			},
			"region": schema.StringAttribute{
				Required:    true,
				Description: "The AWS region the client is present in.",
			},
		},
		SecretsMappingAttributes: map[string]schema.Attribute{
			"access_key_id": schema.StringAttribute{
				Required:    true,
				Description: "The name of the secret that the access key ID will be mapped to.",
			},
			"secret_access_key": schema.StringAttribute{
				Required:    true,
				Description: "The name of the secret that the rotated secret access key will be mapped to.",
			},
		},

		ReadParametersFromPlan: func(ctx context.Context, plan SecretRotationBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			parametersMap := make(map[string]interface{})
			var parameters SecretRotationAwsIAMSecretParametersModel

			diags := plan.Parameters.As(ctx, &parameters, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			parametersMap["userName"] = parameters.UserName.ValueString()
			parametersMap["region"] = parameters.Region.ValueString()

			return parametersMap, diags
		},

		ReadParametersFromApi: func(ctx context.Context, secretRotation infisical.SecretRotation) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics
			parameters := make(map[string]attr.Value)
			parametersSchema := map[string]attr.Type{
				"user_name": types.StringType,
				"region":    types.StringType,
			}

			userNameVal, ok := secretRotation.Parameters["userName"].(string)
			if !ok {
				diags.AddError("API Reading Error", "Expected 'userName' (string) but got wrong type or missing")
				return types.ObjectNull(parametersSchema), diags
			}
			parameters["user_name"] = types.StringValue(userNameVal)

			regionVal, ok := secretRotation.Parameters["region"].(string)
			if !ok {
				diags.AddError("API Reading Error", "Expected 'region' (string) but got wrong type or missing")
				return types.ObjectNull(parametersSchema), diags
			}
			parameters["region"] = types.StringValue(regionVal)

			obj, objDiags := types.ObjectValue(parametersSchema, parameters)
			diags.Append(objDiags...)
			if diags.HasError() {
				return types.ObjectNull(parametersSchema), diags
			}

			return obj, diags
		},

		ReadSecretsMappingFromPlan: func(ctx context.Context, plan SecretRotationBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			secretsMappingMap := make(map[string]interface{})
			var secretsMapping SecretRotationAwsIAMSecretSecretsMappingModel

			diags := plan.SecretsMapping.As(ctx, &secretsMapping, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			secretsMappingMap["accessKeyId"] = secretsMapping.AccessKeyID.ValueString()
			secretsMappingMap["secretAccessKey"] = secretsMapping.SecretAccessKey.ValueString()

			return secretsMappingMap, diags
		},

		ReadSecretsMappingFromApi: func(ctx context.Context, secretRotation infisicalclient.SecretRotation) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics
			secretsMapping := make(map[string]attr.Value)
			secretsMappingSchema := map[string]attr.Type{
				"access_key_id":     types.StringType,
				"secret_access_key": types.StringType,
			}

			accessKeyIdVal, ok := secretRotation.SecretsMapping["accessKeyId"].(string)
			if !ok {
				diags.AddError("API Reading Error", "Expected 'accessKeyId' (string) but got wrong type or missing")
				return types.ObjectNull(secretsMappingSchema), diags
			}
			secretsMapping["access_key_id"] = types.StringValue(accessKeyIdVal)

			SecretAccessKeyVal, ok := secretRotation.SecretsMapping["secretAccessKey"].(string)
			if !ok {
				diags.AddError("API Reading Error", "Expected 'secretAccessKey' (string) but got wrong type or missing")
				return types.ObjectNull(secretsMappingSchema), diags
			}
			secretsMapping["secret_access_key"] = types.StringValue(SecretAccessKeyVal)

			obj, objDiags := types.ObjectValue(secretsMappingSchema, secretsMapping)
			diags.Append(objDiags...)
			if diags.HasError() {
				return types.ObjectNull(secretsMappingSchema), diags
			}

			return obj, diags
		},
	}
}
