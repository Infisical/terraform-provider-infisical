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

// ExternalKmsAwsCredentialsModel describes the credentials data model.
type ExternalKmsAwsCredentialsModel struct {
	AccessKeyId     types.String `tfsdk:"access_key_id"`
	SecretAccessKey types.String `tfsdk:"secret_access_key"`
	RoleARN         types.String `tfsdk:"role_arn"`
	RoleExternalId  types.String `tfsdk:"role_external_id"`
}

// ExternalKmsAwsConfigurationModel describes the configuration data model.
type ExternalKmsAwsConfigurationModel struct {
	AwsRegion   types.String                   `tfsdk:"aws_region"`
	Type        types.String                   `tfsdk:"type"`
	Credential  ExternalKmsAwsCredentialsModel `tfsdk:"credential"`
	AwsKmsKeyId types.String                   `tfsdk:"aws_kms_key_id"`
}

const AwsExternalKmsAccessKeyType = "access-key"
const AwsExternalKmsAssumeRoleType = "assume-role"

func NewExternalKmsAwsResource() resource.Resource {
	return &ExternalKmsBaseResource{
		Provider:                infisical.ExternalKmsProviderAWS,
		ExternalKmsProviderName: "AWS",
		ResourceTypeName:        "_external_kms_aws",
		ConfigurationAttributes: map[string]schema.Attribute{
			"aws_region": schema.StringAttribute{
				Required:    true,
				Description: "The AWS region where the KMS key is located",
			},
			"aws_kms_key_id": schema.StringAttribute{
				Required:    true,
				Description: "The AWS KMS key ID to use for the external KMS. For more details, refer to the documentation here https://infisical.com/docs/documentation/platform/kms-configuration/aws-kms#param-aws-kms-key-id",
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: fmt.Sprintf("The Authentication Type to use. Must be %s or %s", AwsExternalKmsAccessKeyType, AwsExternalKmsAssumeRoleType),
			},
			"credential": schema.SingleNestedAttribute{
				Required:    true,
				Description: "The AWS credentials for the external KMS",
				Attributes: map[string]schema.Attribute{
					"access_key_id": schema.StringAttribute{
						Optional:    true,
						Description: "The AWS Access Key ID used to authenticate requests to AWS services. Required for access-key type. For more details, refer to the documentation here https://infisical.com/docs/documentation/platform/kms-configuration/aws-kms#param-access-key-id",
						Sensitive:   true,
					},
					"secret_access_key": schema.StringAttribute{
						Optional:    true,
						Description: "The AWS Secret Access Key associated with the Access Key ID to authenticate requests to AWS services. Required for access-key type. For more details, refer to the documentation here https://infisical.com/docs/documentation/platform/kms-configuration/aws-kms#param-secret-access-key",
						Sensitive:   true,
					},
					"role_external_id": schema.StringAttribute{
						Optional:    true,
						Description: "The external ID of the role to assume for performing operations. Required for assume-role type. For more details, refer to the documentation here https://infisical.com/docs/documentation/platform/kms-configuration/aws-kms#param-assume-role-external-id",
						Sensitive:   true,
					},
					"role_arn": schema.StringAttribute{
						Optional:    true,
						Description: "The Amazon Resource Name (ARN) of the IAM role to assume for performing operations. Infisical will assume this role using AWS Security Token Service (STS). Required for assume-role type. For more details, refer to the documentation here https://infisical.com/docs/documentation/platform/kms-configuration/aws-kms#param-iam-role-arn-for-role-assumption",
						Sensitive:   true,
					},
				},
			},
		},
		ReadConfigurationForCreateFromPlan: func(ctx context.Context, plan ExternalKmsBaseResourceModel) (map[string]any, diag.Diagnostics) {
			configurationMap := make(map[string]any)
			var diags diag.Diagnostics

			var configuration ExternalKmsAwsConfigurationModel
			diags = plan.Configuration.As(ctx, &configuration, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			credentialData := make(map[string]string)

			if configuration.Type.ValueString() == AwsExternalKmsAccessKeyType {
				if configuration.Credential.AccessKeyId.IsNull() || configuration.Credential.AccessKeyId.ValueString() == "" {
					diags.AddError("Unable to create AWS external KMS", "Access key id field must be defined in access-key type")
					return nil, diags
				}

				if configuration.Credential.SecretAccessKey.IsNull() || configuration.Credential.SecretAccessKey.ValueString() == "" {
					diags.AddError("Unable to create AWS external KMS", "Secret access key field must be defined in access-key type")
					return nil, diags
				}

				credentialData["accessKey"] = configuration.Credential.AccessKeyId.ValueString()
				credentialData["secretKey"] = configuration.Credential.SecretAccessKey.ValueString()
			} else if configuration.Type.ValueString() == AwsExternalKmsAssumeRoleType {
				if configuration.Credential.RoleExternalId.IsNull() || configuration.Credential.RoleExternalId.ValueString() == "" {
					diags.AddError("Unable to create AWS external KMS", "Role external id field must be defined in assume-role type")
					return nil, diags
				}

				if configuration.Credential.RoleARN.IsNull() || configuration.Credential.RoleARN.ValueString() == "" {
					diags.AddError("Unable to create AWS external KMS", "Role arn field must be defined in assume-role type")
					return nil, diags
				}

				credentialData["assumeRoleArn"] = configuration.Credential.RoleARN.ValueString()
				credentialData["externalId"] = configuration.Credential.RoleExternalId.ValueString()
			} else {
				diags.AddError("Invalid Authentication Type", "The 'type' attribute must be 'access-key' or 'assume-role'")
				return nil, diags
			}

			configurationMap["credential"] = map[string]any{
				"type": configuration.Type.ValueString(),
				"data": credentialData,
			}
			configurationMap["awsRegion"] = configuration.AwsRegion.ValueString()
			configurationMap["kmsKeyId"] = configuration.AwsKmsKeyId.ValueString()

			return configurationMap, diags
		},
		ReadConfigurationForUpdateFromPlan: func(ctx context.Context, plan ExternalKmsBaseResourceModel, state ExternalKmsBaseResourceModel) (map[string]any, diag.Diagnostics) {
			configurationMap := make(map[string]any)

			var configurationFromPlan ExternalKmsAwsConfigurationModel
			diags := plan.Configuration.As(ctx, &configurationFromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var configurationFromState ExternalKmsAwsConfigurationModel
			diags = state.Configuration.As(ctx, &configurationFromState, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			credentialData := make(map[string]string)

			if configurationFromPlan.Type.ValueString() == AwsExternalKmsAccessKeyType {
				accessKeyId := configurationFromPlan.Credential.AccessKeyId
				if configurationFromPlan.Credential.AccessKeyId.IsUnknown() {
					accessKeyId = configurationFromState.Credential.AccessKeyId
				}

				secretAccessKey := configurationFromPlan.Credential.SecretAccessKey
				if configurationFromPlan.Credential.SecretAccessKey.IsUnknown() {
					secretAccessKey = configurationFromState.Credential.SecretAccessKey
				}

				if accessKeyId.IsNull() || accessKeyId.ValueString() == "" {
					diags.AddError("Unable to update AWS external KMS", "Access key id field must be defined in access-key type")
					return nil, diags
				}

				if secretAccessKey.IsNull() || secretAccessKey.ValueString() == "" {
					diags.AddError("Unable to update AWS external KMS", "Secret access key field must be defined in access-key type")
					return nil, diags
				}

				credentialData["accessKey"] = accessKeyId.ValueString()
				credentialData["secretKey"] = secretAccessKey.ValueString()
			} else if configurationFromPlan.Type.ValueString() == AwsExternalKmsAssumeRoleType {
				roleExternalId := configurationFromPlan.Credential.RoleExternalId
				if configurationFromPlan.Credential.RoleExternalId.IsUnknown() {
					roleExternalId = configurationFromState.Credential.RoleExternalId
				}

				roleArn := configurationFromPlan.Credential.RoleARN
				if configurationFromPlan.Credential.RoleARN.IsUnknown() {
					roleArn = configurationFromState.Credential.RoleARN
				}

				if roleExternalId.IsNull() || roleExternalId.ValueString() == "" {
					diags.AddError("Unable to update AWS external KMS", "Role external id field must be defined in assume-role type")
					return nil, diags
				}

				if roleArn.IsNull() || roleArn.ValueString() == "" {
					diags.AddError("Unable to update AWS external KMS", "Role arn field must be defined in assume-role type")
					return nil, diags
				}

				credentialData["assumeRoleArn"] = roleArn.ValueString()
				credentialData["externalId"] = roleExternalId.ValueString()
			} else {
				diags.AddError("Invalid Authentication Type", "The 'type' attribute must be 'access-key' or 'assume-role'")
				return nil, diags
			}
			configurationMap["credential"] = map[string]any{
				"type": configurationFromPlan.Type.ValueString(),
				"data": credentialData,
			}
			configurationMap["awsRegion"] = configurationFromPlan.AwsRegion.ValueString()
			configurationMap["kmsKeyId"] = configurationFromPlan.AwsKmsKeyId.ValueString()

			return configurationMap, diags
		},
		OverwriteConfigurationFields: func(state *ExternalKmsBaseResourceModel) diag.Diagnostics {
			var diags diag.Diagnostics

			configAttrs := state.Configuration.Attributes()

			credentialTypes := map[string]attr.Type{
				"access_key_id":     types.StringType,
				"secret_access_key": types.StringType,
				"role_arn":          types.StringType,
				"role_external_id":  types.StringType,
			}

			credentialConfig := map[string]attr.Value{
				"access_key_id":     types.StringNull(),
				"secret_access_key": types.StringNull(),
				"role_arn":          types.StringNull(),
				"role_external_id":  types.StringNull(),
			}

			configAttrs["credential"] = types.ObjectValueMust(
				credentialTypes,
				credentialConfig,
			)

			configTypes := map[string]attr.Type{
				"aws_region":     types.StringType,
				"aws_kms_key_id": types.StringType,
				"type":           types.StringType,
				"credential": types.ObjectType{
					AttrTypes: credentialTypes,
				},
			}

			state.Configuration, diags = types.ObjectValue(configTypes, configAttrs)

			return diags
		},
	}
}
