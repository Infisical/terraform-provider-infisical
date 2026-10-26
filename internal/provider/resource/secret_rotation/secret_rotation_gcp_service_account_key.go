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

type SecretRotationGcpServiceAccountKeyParametersModel struct {
	ServiceAccountEmail types.String `tfsdk:"service_account_email"`
}

type SecretRotationGcpServiceAccountKeySecretsMappingModel struct {
	ServiceAccountKey types.String `tfsdk:"service_account_key"`
}

func NewSecretRotationGcpServiceAccountKeyResource() resource.Resource {
	return &SecretRotationBaseResource{
		Provider:           infisical.SecretRotationProviderGcpServiceAccountKey,
		SecretRotationName: "GCP Service Account Key",
		ResourceTypeName:   "_secret_rotation_gcp_service_account_key",
		AppConnection:      infisical.AppConnectionAppGCP,
		ParametersAttributes: map[string]schema.Attribute{
			"service_account_email": schema.StringAttribute{
				Required:    true,
				Description: "The email of the GCP service account whose keys will be rotated, e.g. my-app@my-project.iam.gserviceaccount.com. The connection's service account needs the Service Account Key Admin role (roles/iam.serviceAccountKeyAdmin) on it.",
			},
		},
		SecretsMappingAttributes: map[string]schema.Attribute{
			"service_account_key": schema.StringAttribute{
				Required:    true,
				Description: "The name of the secret that the rotated service account key (the JSON key file) will be mapped to.",
			},
		},

		ReadParametersFromPlan: func(ctx context.Context, plan SecretRotationBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			parametersMap := make(map[string]interface{})
			var parameters SecretRotationGcpServiceAccountKeyParametersModel

			diags := plan.Parameters.As(ctx, &parameters, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			parametersMap["serviceAccountEmail"] = parameters.ServiceAccountEmail.ValueString()

			return parametersMap, diags
		},

		ReadParametersFromApi: func(ctx context.Context, secretRotation infisical.SecretRotation) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics
			parameters := make(map[string]attr.Value)
			parametersSchema := map[string]attr.Type{
				"service_account_email": types.StringType,
			}

			serviceAccountEmailVal, ok := secretRotation.Parameters["serviceAccountEmail"].(string)
			if !ok {
				diags.AddError("API Reading Error", "Expected 'serviceAccountEmail' (string) but got wrong type or missing")
				return types.ObjectNull(parametersSchema), diags
			}
			parameters["service_account_email"] = types.StringValue(serviceAccountEmailVal)

			obj, objDiags := types.ObjectValue(parametersSchema, parameters)
			diags.Append(objDiags...)
			if diags.HasError() {
				return types.ObjectNull(parametersSchema), diags
			}

			return obj, diags
		},

		ReadSecretsMappingFromPlan: func(ctx context.Context, plan SecretRotationBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			secretsMappingMap := make(map[string]interface{})
			var secretsMapping SecretRotationGcpServiceAccountKeySecretsMappingModel

			diags := plan.SecretsMapping.As(ctx, &secretsMapping, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			secretsMappingMap["serviceAccountKey"] = secretsMapping.ServiceAccountKey.ValueString()

			return secretsMappingMap, diags
		},

		ReadSecretsMappingFromApi: func(ctx context.Context, secretRotation infisical.SecretRotation) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics
			secretsMapping := make(map[string]attr.Value)
			secretsMappingSchema := map[string]attr.Type{
				"service_account_key": types.StringType,
			}

			serviceAccountKeyVal, ok := secretRotation.SecretsMapping["serviceAccountKey"].(string)
			if !ok {
				diags.AddError("API Reading Error", "Expected 'serviceAccountKey' (string) but got wrong type or missing")
				return types.ObjectNull(secretsMappingSchema), diags
			}
			secretsMapping["service_account_key"] = types.StringValue(serviceAccountKeyVal)

			obj, objDiags := types.ObjectValue(secretsMappingSchema, secretsMapping)
			diags.Append(objDiags...)
			if diags.HasError() {
				return types.ObjectNull(secretsMappingSchema), diags
			}

			return obj, diags
		},
	}
}
