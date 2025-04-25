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

// AppConnectionAwsCredentialsModel describes the data source data model.
type AppConnectionAwsCredentialsModel struct {
	RoleARN         types.String `tfsdk:"role_arn"`
	AccessKeyId     types.String `tfsdk:"access_key_id"`
	SecretAccessKey types.String `tfsdk:"secret_access_key"`
}

const AwsAppConnectionAssumeRoleMethod = "assume-role"
const AwsAppConnectionAccessKeyMethod = "access-key"

func NewAppConnectionAwsResource() resource.Resource {
	return &AppConnectionBaseResource{
		App:               infisical.AppConnectionAppAWS,
		AppConnectionName: "AWS",
		ResourceTypeName:  "_app_connection_aws",
		AllowedMethods:    []string{AwsAppConnectionAssumeRoleMethod, AwsAppConnectionAccessKeyMethod},
		CredentialsAttributes: map[string]schema.Attribute{
			"role_arn": schema.StringAttribute{
				Optional:    true,
				Description: "The Amazon Resource Name (ARN) of the IAM role to assume for performing operations. Infisical will assume this role using AWS Security Token Service (STS). Required for assume-role access method. For more details, refer to the documentation here infisical.com/docs/integrations/app-connections/aws#assume-role-recommended",
				Sensitive:   true,
			},
			"access_key_id": schema.StringAttribute{
				Optional:    true,
				Description: "The AWS Access Key ID used to authenticate requests to AWS services. Required for access-key access method. For more details, refer to the documentation here infisical.com/docs/integrations/app-connections/aws#access-key",
				Sensitive:   true,
			},
			"secret_access_key": schema.StringAttribute{
				Optional:    true,
				Description: "The AWS Secret Access Key associated with the Access Key ID to authenticate requests to AWS services. Required for access-key access method. For more details, refer to the documentation here infisical.com/docs/integrations/app-connections/aws#access-key",
				Sensitive:   true,
			},
		},
		ReadCredentialsForCreateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			credentialsConfig := make(map[string]interface{})

			var credentials AppConnectionAwsCredentialsModel
			diags := plan.Credentials.As(ctx, &credentials, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() == AwsAppConnectionAssumeRoleMethod {
				if credentials.RoleARN.IsNull() || credentials.RoleARN.ValueString() == "" {
					diags.AddError(
						"Unable to create AWS app connection",
						"Role arn field must be defined in assume-role method",
					)
					return nil, diags
				}

				credentialsConfig["roleArn"] = credentials.RoleARN.ValueString()
			} else {
				if credentials.AccessKeyId.IsNull() || credentials.AccessKeyId.ValueString() == "" {
					diags.AddError(
						"Unable to create AWS app connection",
						"Access key id field must be defined in access-key method",
					)
					return nil, diags
				}

				if credentials.SecretAccessKey.IsNull() || credentials.SecretAccessKey.ValueString() == "" {
					diags.AddError(
						"Unable to create AWS app connection",
						"Secret access key field must be defined in access-key method",
					)
					return nil, diags
				}

				credentialsConfig["accessKeyId"] = credentials.AccessKeyId.ValueString()
				credentialsConfig["secretAccessKey"] = credentials.SecretAccessKey.ValueString()
			}

			return credentialsConfig, diags
		},
		ReadCredentialsForUpdateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel, state AppConnectionBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			credentialsConfig := make(map[string]interface{})

			var credentialsFromPlan AppConnectionAwsCredentialsModel
			diags := plan.Credentials.As(ctx, &credentialsFromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var credentialsFromState AppConnectionAwsCredentialsModel
			diags = state.Credentials.As(ctx, &credentialsFromState, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() == AwsAppConnectionAssumeRoleMethod {
				if credentialsFromPlan.RoleARN.IsNull() || credentialsFromPlan.RoleARN.ValueString() == "" {
					diags.AddError(
						"Unable to update AWS app connection",
						"Role arn field must be defined in assume-role method",
					)
					return nil, diags
				}

				if credentialsFromState.RoleARN.ValueString() != credentialsFromPlan.RoleARN.ValueString() {
					credentialsConfig["roleArn"] = credentialsFromPlan.RoleARN.ValueString()
				}
			} else {
				if credentialsFromPlan.AccessKeyId.IsNull() || credentialsFromPlan.AccessKeyId.ValueString() == "" {
					diags.AddError(
						"Unable to update AWS app connection",
						"Access key id field must be defined in access-key method",
					)
					return nil, diags
				}

				if credentialsFromPlan.SecretAccessKey.IsNull() || credentialsFromPlan.SecretAccessKey.ValueString() == "" {
					diags.AddError(
						"Unable to update AWS app connection",
						"Secret access key field must be defined in access-key method",
					)
					return nil, diags
				}

				credentialsConfig["accessKeyId"] = credentialsFromPlan.AccessKeyId.ValueString()
				credentialsConfig["secretAccessKey"] = credentialsFromPlan.SecretAccessKey.ValueString()
			}

			return credentialsConfig, diags
		},
		OverwriteCredentialsFields: func(state *AppConnectionBaseResourceModel) diag.Diagnostics {
			credentialsConfig := map[string]attr.Value{
				"role_arn":          types.StringNull(),
				"access_key_id":     types.StringNull(),
				"secret_access_key": types.StringNull(),
			}

			var diags diag.Diagnostics
			state.Credentials, diags = types.ObjectValue(map[string]attr.Type{
				"role_arn":          types.StringType,
				"access_key_id":     types.StringType,
				"secret_access_key": types.StringType,
			}, credentialsConfig)

			return diags
		},
	}
}
