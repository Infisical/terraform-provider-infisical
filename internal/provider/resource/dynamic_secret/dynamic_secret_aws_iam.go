package resource

import (
	"context"
	infisicalclient "terraform-provider-infisical/internal/client"
	pkg "terraform-provider-infisical/internal/pkg/modifiers"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type AccessKeyConfigurationModel struct {
	AccessKey       types.String `tfsdk:"access_key"`
	SecretAccessKey types.String `tfsdk:"secret_access_key"`
}

type AssumeRoleConfigurationModel struct {
	RoleArn types.String `tfsdk:"role_arn"`
}

type DynamicSecretAwsIamConfigurationModel struct {
	Method                      types.String                  `tfsdk:"method"`
	AccessKeyConfig             *AccessKeyConfigurationModel  `tfsdk:"access_key_config"`
	AssumeRoleConfig            *AssumeRoleConfigurationModel `tfsdk:"assume_role_config"`
	Region                      types.String                  `tfsdk:"region"`
	AwsPath                     types.String                  `tfsdk:"aws_path"`
	PermissionBoundaryPolicyArn types.String                  `tfsdk:"permission_boundary_policy_arn"`
	PolicyDocument              types.String                  `tfsdk:"policy_document"`
	UserGroups                  types.String                  `tfsdk:"user_groups"`
	PolicyArns                  types.String                  `tfsdk:"policy_arns"`
}

func NewDynamicSecretAwsIamResource() resource.Resource {
	return &DynamicSecretBaseResource{
		Provider:          infisicalclient.DynamicSecretProviderAWSIAM,
		ResourceTypeName:  "_dynamic_secret_aws_iam",
		DynamicSecretName: "AWS IAM",
		ConfigurationAttributes: map[string]schema.Attribute{
			"method": schema.StringAttribute{
				Required:    true,
				Description: "The authentication method to use. Must be 'access_key' or 'assume_role'.",
			},
			"access_key_config": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Configuration for the 'access_key' authentication method.",
				Attributes: map[string]schema.Attribute{
					"access_key": schema.StringAttribute{
						Required:    true,
						Description: "The managing AWS IAM User Access Key",
					},
					"secret_access_key": schema.StringAttribute{
						Required:    true,
						Description: "The managing AWS IAM User Secret Key",
						Sensitive:   true,
					},
				},
			},
			"assume_role_config": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Configuration for the 'assume_role' authentication method.",
				Attributes: map[string]schema.Attribute{
					"role_arn": schema.StringAttribute{
						Required:    true,
						Description: "The ARN of the AWS Role to assume.",
					},
				},
			},
			"region": schema.StringAttribute{
				Required:    true,
				Description: "The AWS data center region.",
			},
			"aws_path": schema.StringAttribute{
				Optional:    true,
				Description: "IAM AWS Path to scope created IAM User resource access.",
			},
			"permission_boundary_policy_arn": schema.StringAttribute{
				Optional:    true,
				Description: "The IAM Policy ARN of the AWS Permissions Boundary to attach to IAM users created in the role.",
			},
			"policy_document": schema.StringAttribute{
				Optional:    true,
				Description: "The AWS IAM inline policy that should be attached to the created users. Multiple values can be provided by separating them with commas",
				PlanModifiers: []planmodifier.String{
					pkg.TrimEqualityModifier{},
				},
			},
			"user_groups": schema.StringAttribute{
				Optional:    true,
				Description: "The AWS IAM groups that should be assigned to the created users. Multiple values can be provided by separating them with commas",
			},
			"policy_arns": schema.StringAttribute{
				Optional:    true,
				Description: "The AWS IAM managed policies that should be attached to the created users. Multiple values can be provided by separating them with commas",
			},
		},

		ReadConfigurationFromPlan: func(ctx context.Context, plan DynamicSecretBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			configurationMap := make(map[string]interface{})
			var configuration DynamicSecretAwsIamConfigurationModel

			diags := plan.Configuration.As(ctx, &configuration, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			method := configuration.Method.ValueString()
			apiMethod := ""
			switch method {
			case "access_key":
				apiMethod = "access-key"
			case "assume_role":
				apiMethod = "assume-role"
			default:
				diags.AddError("Invalid Configuration Method", "The 'method' attribute must be 'access_key' or 'assume_role'.")
				return nil, diags
			}
			configurationMap["method"] = apiMethod

			switch method {
			case "access_key":
				if configuration.AccessKeyConfig == nil {
					diags.AddError(
						"Invalid Configuration",
						"When method is 'access_key', the 'access_key_config' block must be provided.",
					)
					return nil, diags
				}
				configurationMap["accessKey"] = configuration.AccessKeyConfig.AccessKey.ValueString()
				configurationMap["secretAccessKey"] = configuration.AccessKeyConfig.SecretAccessKey.ValueString()
			case "assume_role":
				if configuration.AssumeRoleConfig == nil {
					diags.AddError(
						"Invalid Configuration",
						"When method is 'assume_role', the 'assume_role_config' block must be provided.",
					)
					return nil, diags
				}
				configurationMap["roleArn"] = configuration.AssumeRoleConfig.RoleArn.ValueString()
			}

			configurationMap["region"] = configuration.Region.ValueString()
			if !configuration.AwsPath.IsNull() {
				configurationMap["awsPath"] = configuration.AwsPath.ValueString()
			}
			if !configuration.PermissionBoundaryPolicyArn.IsNull() {
				configurationMap["permissionBoundaryPolicyArn"] = configuration.PermissionBoundaryPolicyArn.ValueString()
			}
			if !configuration.PolicyDocument.IsNull() {
				configurationMap["policyDocument"] = configuration.PolicyDocument.ValueString()
			}
			if !configuration.UserGroups.IsNull() {
				configurationMap["userGroups"] = configuration.UserGroups.ValueString()
			}
			if !configuration.PolicyArns.IsNull() {
				configurationMap["policyArns"] = configuration.PolicyArns.ValueString()
			}

			return configurationMap, diags
		},

		ReadConfigurationFromApi: func(ctx context.Context, dynamicSecret infisicalclient.DynamicSecret, configState types.Object) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics
			configuration := make(map[string]attr.Value)
			configurationSchema := map[string]attr.Type{
				"method": types.StringType,
				"access_key_config": types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"access_key":        types.StringType,
						"secret_access_key": types.StringType,
					},
				},
				"assume_role_config": types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"role_arn": types.StringType,
					},
				},
				"region":                         types.StringType,
				"aws_path":                       types.StringType,
				"permission_boundary_policy_arn": types.StringType,
				"policy_document":                types.StringType,
				"user_groups":                    types.StringType,
				"policy_arns":                    types.StringType,
			}

			// Read API method value (expected to be "access-key" or "assume-role")
			apiMethodVal, ok := dynamicSecret.Inputs["method"].(string)
			tfMethod := ""
			if ok && apiMethodVal != "" {
				switch apiMethodVal {
				case "access-key":
					tfMethod = "access_key"
				case "assume-role":
					tfMethod = "assume_role"
				default:
					diags.AddError("API Reading Error", "Unknown AWS IAM method from API: "+apiMethodVal)
					return types.ObjectNull(configurationSchema), diags
				}
			} else {
				// Default to access_key if method is missing or empty in API response
				tfMethod = "access_key"
			}
			configuration["method"] = types.StringValue(tfMethod)

			accessKeyConfigSchemaEntry, okAks := configurationSchema["access_key_config"].(types.ObjectType)
			if !okAks {
				diags.AddError(
					"Internal Schema Error",
					"Could not assert 'access_key_config' to types.ObjectType. This indicates an issue with the provider's internal schema definition.",
				)
				return types.ObjectNull(configurationSchema), diags
			}
			configuration["access_key_config"] = types.ObjectNull(accessKeyConfigSchemaEntry.AttrTypes)
			assumeRoleConfigSchemaEntry, okArs := configurationSchema["assume_role_config"].(types.ObjectType)
			if !okArs {
				diags.AddError(
					"Internal Schema Error",
					"Could not assert 'assume_role_config' to types.ObjectType. This indicates an issue with the provider's internal schema definition.",
				)
				return types.ObjectNull(configurationSchema), diags
			}
			configuration["assume_role_config"] = types.ObjectNull(assumeRoleConfigSchemaEntry.AttrTypes)

			switch tfMethod {
			case "access_key":
				accessKeyConfigMap := make(map[string]attr.Value)
				accessKeyConfigAttrType, ok := configurationSchema["access_key_config"].(types.ObjectType)
				if !ok {
					diags.AddError(
						"Internal Schema Error",
						"Could not assert 'access_key_config' to types.ObjectType within access_key case. This indicates an issue with the provider's internal schema definition.",
					)
					return types.ObjectNull(configurationSchema), diags
				}
				accessKeyConfigSchema := accessKeyConfigAttrType.AttrTypes

				accessKeyVal, ok := dynamicSecret.Inputs["accessKey"].(string)
				if !ok {
					diags.AddError("API Reading Error", "Expected 'accessKey' (string) but got wrong type or missing")
					return types.ObjectNull(configurationSchema), diags
				}
				accessKeyConfigMap["access_key"] = types.StringValue(accessKeyVal)

				secretAccessKeyVal, ok := dynamicSecret.Inputs["secretAccessKey"].(string)
				if !ok {
					diags.AddError("API Reading Error", "Expected 'secretAccessKey' (string) but got wrong type or missing")
					return types.ObjectNull(configurationSchema), diags
				}
				accessKeyConfigMap["secret_access_key"] = types.StringValue(secretAccessKeyVal)

				accessKeyConfigObj, accessKeyConfigDiags := types.ObjectValue(accessKeyConfigSchema, accessKeyConfigMap)
				diags.Append(accessKeyConfigDiags...)
				if diags.HasError() {
					return types.ObjectNull(configurationSchema), diags
				}
				configuration["access_key_config"] = accessKeyConfigObj

				regionVal, ok := dynamicSecret.Inputs["region"].(string)
				if !ok {
					diags.AddError("API Reading Error", "Expected 'region' (string) but got wrong type or missing")
					return types.ObjectNull(configurationSchema), diags
				}
				configuration["region"] = types.StringValue(regionVal)

				awsPathVal, ok := dynamicSecret.Inputs["awsPath"].(string)
				awsPathValue := types.StringNull()
				if ok && awsPathVal != "" {
					awsPathValue = types.StringValue(awsPathVal)
				}
				configuration["aws_path"] = awsPathValue

				permissionBoundaryPolicyArnVal, ok := dynamicSecret.Inputs["permissionBoundaryPolicyArn"].(string)
				permissionBoundaryPolicyArnValue := types.StringNull()
				if ok && permissionBoundaryPolicyArnVal != "" {
					permissionBoundaryPolicyArnValue = types.StringValue(permissionBoundaryPolicyArnVal)
				}
				configuration["permission_boundary_policy_arn"] = permissionBoundaryPolicyArnValue

				policyDocumentVal, ok := dynamicSecret.Inputs["policyDocument"].(string)
				policyDocumentValue := types.StringNull()
				if ok && policyDocumentVal != "" {
					policyDocumentValue = types.StringValue(policyDocumentVal)
				}
				configuration["policy_document"] = policyDocumentValue

				userGroupsVal, ok := dynamicSecret.Inputs["userGroups"].(string)
				userGroupsValue := types.StringNull()
				if ok && userGroupsVal != "" {
					userGroupsValue = types.StringValue(userGroupsVal)
				}
				configuration["user_groups"] = userGroupsValue

				policyArnsVal, ok := dynamicSecret.Inputs["policyArns"].(string)
				policyArnsValue := types.StringNull()
				if ok && policyArnsVal != "" {
					policyArnsValue = types.StringValue(policyArnsVal)
				}
				configuration["policy_arns"] = policyArnsValue

			case "assume_role":
				assumeRoleConfigMap := make(map[string]attr.Value)
				assumeRoleConfigAttrType, ok := configurationSchema["assume_role_config"].(types.ObjectType)
				if !ok {
					diags.AddError(
						"Internal Schema Error",
						"Could not assert 'assume_role_config' to types.ObjectType within assume_role case. This indicates an issue with the provider's internal schema definition.",
					)
					return types.ObjectNull(configurationSchema), diags
				}
				assumeRoleConfigSchema := assumeRoleConfigAttrType.AttrTypes

				roleArnVal, ok := dynamicSecret.Inputs["roleArn"].(string)
				if !ok {
					diags.AddError("API Reading Error", "Expected 'roleArn' (string) but got wrong type or missing")
					return types.ObjectNull(configurationSchema), diags
				}
				assumeRoleConfigMap["role_arn"] = types.StringValue(roleArnVal)

				assumeRoleConfigObj, assumeRoleConfigDiags := types.ObjectValue(assumeRoleConfigSchema, assumeRoleConfigMap)
				diags.Append(assumeRoleConfigDiags...)
				if diags.HasError() {
					return types.ObjectNull(configurationSchema), diags
				}
				configuration["assume_role_config"] = assumeRoleConfigObj

				regionVal, ok := dynamicSecret.Inputs["region"].(string)
				if !ok {
					diags.AddError("API Reading Error", "Expected 'region' (string) but got wrong type or missing")
					return types.ObjectNull(configurationSchema), diags
				}
				configuration["region"] = types.StringValue(regionVal)

				awsPathVal, ok := dynamicSecret.Inputs["awsPath"].(string)
				awsPathValue := types.StringNull()
				if ok && awsPathVal != "" {
					awsPathValue = types.StringValue(awsPathVal)
				}
				configuration["aws_path"] = awsPathValue

				permissionBoundaryPolicyArnVal, ok := dynamicSecret.Inputs["permissionBoundaryPolicyArn"].(string)
				permissionBoundaryPolicyArnValue := types.StringNull()
				if ok && permissionBoundaryPolicyArnVal != "" {
					permissionBoundaryPolicyArnValue = types.StringValue(permissionBoundaryPolicyArnVal)
				}
				configuration["permission_boundary_policy_arn"] = permissionBoundaryPolicyArnValue

				policyDocumentVal, ok := dynamicSecret.Inputs["policyDocument"].(string)
				policyDocumentValue := types.StringNull()
				if ok && policyDocumentVal != "" {
					policyDocumentValue = types.StringValue(policyDocumentVal)
				}
				configuration["policy_document"] = policyDocumentValue

				userGroupsVal, ok := dynamicSecret.Inputs["userGroups"].(string)
				userGroupsValue := types.StringNull()
				if ok && userGroupsVal != "" {
					userGroupsValue = types.StringValue(userGroupsVal)
				}
				configuration["user_groups"] = userGroupsValue

				policyArnsVal, ok := dynamicSecret.Inputs["policyArns"].(string)
				policyArnsValue := types.StringNull()
				if ok && policyArnsVal != "" {
					policyArnsValue = types.StringValue(policyArnsVal)
				}
				configuration["policy_arns"] = policyArnsValue

			default:
				diags.AddError("API Reading Error", "Internal Error: Mapped unknown AWS IAM method: "+tfMethod)
				return types.ObjectNull(configurationSchema), diags
			}

			obj, objDiags := types.ObjectValue(configurationSchema, configuration)
			diags.Append(objDiags...)
			if diags.HasError() {
				return types.ObjectNull(configurationSchema), diags
			}

			return obj, diags
		},
	}
}
