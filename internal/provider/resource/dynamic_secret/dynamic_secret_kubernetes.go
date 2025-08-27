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

type ApiConfigurationModel struct {
	ClusterUrl   types.String `tfsdk:"cluster_url"`
	ClusterToken types.String `tfsdk:"cluster_token"`

	EnableSsl types.Bool   `tfsdk:"enable_ssl"`
	Ca        types.String `tfsdk:"ca"`
}

type StaticConfigurationModel struct {
	ServiceAccountName types.String `tfsdk:"service_account_name"`
	Namespace          types.String `tfsdk:"namespace"`
}

type DynamicConfigurationModel struct {
	AllowedNamespaces types.String `tfsdk:"allowed_namespaces"`
	Role              types.String `tfsdk:"role"`
	RoleType          types.String `tfsdk:"role_type"` // "cluster-role" | "role"
}

type DynamicSecretKubernetesConfigurationModel struct {
	GatewayId types.String `tfsdk:"gateway_id"`

	AuthMethod types.String           `tfsdk:"auth_method"` // "api" | "gateway"
	ApiConfig  *ApiConfigurationModel `tfsdk:"api_config"`

	CredentialType types.String               `tfsdk:"credential_type"` // "static" | "dynamic"
	StaticConfig   *StaticConfigurationModel  `tfsdk:"static_config"`
	DynamicConfig  *DynamicConfigurationModel `tfsdk:"dynamic_config"`

	Audiences types.List `tfsdk:"audiences"`
}

func NewDynamicSecretKubernetesResource() resource.Resource {
	return &DynamicSecretBaseResource{
		Provider:          infisical.DynamicSecretProviderKubernetes,
		ResourceTypeName:  "_dynamic_secret_kubernetes",
		DynamicSecretName: "Kubernetes",
		ConfigurationAttributes: map[string]schema.Attribute{
			"gateway_id": schema.StringAttribute{
				Optional:    true,
				Description: "Select a gateway for private cluster access. If not specified, the Internet Gateway will be used.",
			},

			"auth_method": schema.StringAttribute{
				Required:    true,
				Description: "Choose between Token ('api') or 'gateway' authentication. If using Gateway, the Gateway must be deployed in your Kubernetes cluster.",
			},
			"api_config": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Configuration for the 'api' authentication method.",
				Attributes: map[string]schema.Attribute{
					"cluster_url": schema.StringAttribute{
						Required:    true,
						Description: "Kubernetes API server URL (e.g., https://kubernetes.default.svc).",
					},
					"cluster_token": schema.StringAttribute{
						Required:    true,
						Description: "Service account token with permissions to create service accounts and manage RBAC.",
						Sensitive:   true,
					},
					"enable_ssl": schema.BoolAttribute{
						Optional:    true,
						Description: "Whether to enable SSL verification for the Kubernetes API server connection.",
					},
					"ca": schema.StringAttribute{
						Optional:    true,
						Description: "Custom CA certificate for the Kubernetes API server. Leave blank to use the system/public CA.",
					},
				},
			},

			"credential_type": schema.StringAttribute{
				Required:    true,
				Description: "Choose between 'static' (predefined service account) or 'dynamic' (temporary service accounts with role assignments).",
			},
			"static_config": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Configuration for the 'static' credential type.",
				Attributes: map[string]schema.Attribute{
					"service_account_name": schema.StringAttribute{
						Required:    true,
						Description: "Name of the service account to generate tokens for.",
					},
					"namespace": schema.StringAttribute{
						Required:    true,
						Description: "Kubernetes namespace where the service account exists.",
					},
				},
			},
			"dynamic_config": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Configuration for the 'dynamic' credential type.",
				Attributes: map[string]schema.Attribute{
					"allowed_namespaces": schema.StringAttribute{
						Required:    true,
						Description: "Kubernetes namespace(s) where the service accounts will be created. You can specify multiple namespaces as a comma-separated list (e.g., “default,kube-system”). During lease creation, you can specify which namespace to use from this allowed list.",
					},
					"role": schema.StringAttribute{
						Required:    true,
						Description: "Name of the role to assign to the temporary service account.",
					},
					"role_type": schema.StringAttribute{
						Required:    true,
						Description: "Type of role to assign ('cluster-role' or 'role').",
					},
				},
			},

			"audiences": schema.ListAttribute{
				Optional:    true,
				Description: "Optional list of audiences to include in the generated token.",
				ElementType: types.StringType,
			},
		},

		ReadConfigurationFromPlan: func(ctx context.Context, plan DynamicSecretBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			configurationMap := make(map[string]interface{})
			var configuration DynamicSecretKubernetesConfigurationModel

			diags := plan.Configuration.As(ctx, &configuration, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if !configuration.GatewayId.IsNull() && !configuration.GatewayId.IsUnknown() {
				configurationMap["gatewayId"] = configuration.GatewayId.ValueString()
			}

			configurationMap["authMethod"] = configuration.AuthMethod.ValueString()

			switch configuration.AuthMethod.ValueString() {
			case "api":
				if configuration.ApiConfig != nil {
					configurationMap["url"] = configuration.ApiConfig.ClusterUrl.ValueString()
					configurationMap["clusterToken"] = configuration.ApiConfig.ClusterToken.ValueString()

					if !configuration.ApiConfig.EnableSsl.IsNull() && !configuration.ApiConfig.EnableSsl.IsUnknown() {
						configurationMap["sslEnabled"] = configuration.ApiConfig.EnableSsl.ValueBool()
					}
					if !configuration.ApiConfig.Ca.IsNull() && !configuration.ApiConfig.Ca.IsUnknown() {
						configurationMap["ca"] = configuration.ApiConfig.Ca.ValueString()
					}
				} else {
					diags.AddError(
						"Invalid Configuration",
						"When auth_method is 'api', the 'api_config' block must be provided.",
					)
					return nil, diags
				}
			case "gateway":
				if configuration.GatewayId.IsNull() || configuration.GatewayId.IsUnknown() || configuration.GatewayId.ValueString() == "" {
					diags.AddError(
						"Invalid Configuration",
						"When auth_method is 'gateway', 'gateway_id' must be provided.",
					)
					return nil, diags
				}
			default:
				diags.AddError(
					"Invalid Configuration",
					"Invalid 'auth_method' value. Must be 'api' or 'gateway'.",
				)
				return nil, diags
			}

			configurationMap["credentialType"] = configuration.CredentialType.ValueString()

			switch configuration.CredentialType.ValueString() {
			case "static":
				if configuration.StaticConfig != nil {
					configurationMap["serviceAccountName"] = configuration.StaticConfig.ServiceAccountName.ValueString()
					configurationMap["namespace"] = configuration.StaticConfig.Namespace.ValueString()
				} else {
					diags.AddError(
						"Invalid Configuration",
						"When credential_type is 'static', the 'static_config' block must be provided.",
					)
					return nil, diags
				}
			case "dynamic":
				if configuration.DynamicConfig != nil {
					configurationMap["namespace"] = configuration.DynamicConfig.AllowedNamespaces.ValueString()
					configurationMap["role"] = configuration.DynamicConfig.Role.ValueString()
					configurationMap["roleType"] = configuration.DynamicConfig.RoleType.ValueString()
				} else {
					diags.AddError(
						"Invalid Configuration",
						"When credential_type is 'dynamic', the 'dynamic_config' block must be provided.",
					)
					return nil, diags
				}
			default:
				diags.AddError(
					"Invalid Configuration",
					"Invalid 'credential_type' value. Must be 'static' or 'dynamic'.",
				)
				return nil, diags
			}

			var audiences []string

			if configuration.Audiences.IsNull() || configuration.Audiences.IsUnknown() {
				audiences = []string{}
			} else {
				listDiags := configuration.Audiences.ElementsAs(ctx, &audiences, false)
				diags.Append(listDiags...)
			}

			configurationMap["audiences"] = audiences

			return configurationMap, diags
		},

		ReadConfigurationFromApi: func(ctx context.Context, dynamicSecret infisical.DynamicSecret, configState types.Object) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			var currentState DynamicSecretKubernetesConfigurationModel
			stateDiags := configState.As(ctx, &currentState, basetypes.ObjectAsOptions{})
			diags.Append(stateDiags...)

			gatewayId := types.StringNull()
			if gatewayIdVal, ok := dynamicSecret.Inputs["gatewayId"].(string); ok {
				gatewayId = types.StringValue(gatewayIdVal)
			}

			authMethod, ok := dynamicSecret.Inputs["authMethod"].(string)
			if !ok {
				diags.AddError(
					"Invalid authMethod type",
					"Expected 'authMethod' to be a string but got something else.",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			credentialType, ok := dynamicSecret.Inputs["credentialType"].(string)
			if !ok {
				diags.AddError(
					"Invalid credentialType type",
					"Expected 'credentialType' to be a string but got something else.",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			configuration := map[string]attr.Value{
				"gateway_id":      gatewayId,
				"auth_method":     types.StringValue(authMethod),
				"credential_type": types.StringValue(credentialType),
			}

			configuration["api_config"] = types.ObjectNull(map[string]attr.Type{
				"cluster_url":   types.StringType,
				"cluster_token": types.StringType,
				"enable_ssl":    types.BoolType,
				"ca":            types.StringType,
			})
			configuration["static_config"] = types.ObjectNull(map[string]attr.Type{
				"service_account_name": types.StringType,
				"namespace":            types.StringType,
			})
			configuration["dynamic_config"] = types.ObjectNull(map[string]attr.Type{
				"allowed_namespaces": types.StringType,
				"role":               types.StringType,
				"role_type":          types.StringType,
			})

			switch authMethod {
			case "api":
				clusterUrl, ok := dynamicSecret.Inputs["url"].(string)
				if !ok {
					diags.AddError(
						"Invalid cluster url type",
						"Expected 'url' to be a string but got something else.",
					)
					return types.ObjectNull(map[string]attr.Type{}), diags
				}

				clusterToken, _ := dynamicSecret.Inputs["clusterToken"].(string)
				enableSsl, _ := dynamicSecret.Inputs["sslEnabled"].(bool)
				ca, _ := dynamicSecret.Inputs["ca"].(string)

				apiConfigMap := map[string]attr.Value{
					"cluster_url":   types.StringValue(clusterUrl),
					"cluster_token": types.StringValue(clusterToken),
					"enable_ssl":    types.BoolValue(enableSsl),
					"ca":            types.StringValue(ca),
				}

				apiConfigObj, apiConfigDiags := types.ObjectValue(map[string]attr.Type{
					"cluster_url":   types.StringType,
					"cluster_token": types.StringType,
					"enable_ssl":    types.BoolType,
					"ca":            types.StringType,
				}, apiConfigMap)
				if apiConfigDiags.HasError() {
					diags.Append(apiConfigDiags...)
					return types.ObjectNull(map[string]attr.Type{}), diags
				}
				configuration["api_config"] = apiConfigObj
			case "gateway":
			default:
				diags.AddError(
					"Invalid authMethod value",
					"Expected 'api' or 'gateway' but got something else.",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			switch credentialType {
			case "static":
				serviceAccountName, _ := dynamicSecret.Inputs["serviceAccountName"].(string)
				namespace, _ := dynamicSecret.Inputs["namespace"].(string)

				staticConfigMap := map[string]attr.Value{
					"service_account_name": types.StringValue(serviceAccountName),
					"namespace":            types.StringValue(namespace),
				}

				staticConfigObj, staticConfigDiags := types.ObjectValue(map[string]attr.Type{
					"service_account_name": types.StringType,
					"namespace":            types.StringType,
				}, staticConfigMap)
				if staticConfigDiags.HasError() {
					diags.Append(staticConfigDiags...)
					return types.ObjectNull(map[string]attr.Type{}), diags
				}
				configuration["static_config"] = staticConfigObj
			case "dynamic":
				namespace, _ := dynamicSecret.Inputs["namespace"].(string)
				role, _ := dynamicSecret.Inputs["role"].(string)
				roleType, _ := dynamicSecret.Inputs["roleType"].(string)

				dynamicConfigMap := map[string]attr.Value{
					"allowed_namespaces": types.StringValue(namespace),
					"role":               types.StringValue(role),
					"role_type":          types.StringValue(roleType),
				}

				dynamicConfigObj, dynamicConfigDiags := types.ObjectValue(map[string]attr.Type{
					"allowed_namespaces": types.StringType,
					"role":               types.StringType,
					"role_type":          types.StringType,
				}, dynamicConfigMap)
				if dynamicConfigDiags.HasError() {
					diags.Append(dynamicConfigDiags...)
					return types.ObjectNull(map[string]attr.Type{}), diags
				}
				configuration["dynamic_config"] = dynamicConfigObj
			default:
				diags.AddError(
					"Invalid credentialType value",
					"Expected 'static' or 'dynamic' but got something else.",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			audiencesRaw, ok := dynamicSecret.Inputs["audiences"].([]any)
			if !ok {
				diags.AddError(
					"Invalid audiences type",
					"Expected 'audiences' to be a list but got something else.",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			plannedListIsUndefinedOrEmpty := currentState.Audiences.IsNull() || len(currentState.Audiences.Elements()) == 0

			if len(audiencesRaw) == 0 && plannedListIsUndefinedOrEmpty {
				configuration["audiences"] = currentState.Audiences
			} else {
				var audiences []string
				for i, v := range audiencesRaw {
					s, ok := v.(string)
					if !ok {
						diags.AddError(
							"Invalid audience element type",
							"Expected audience at index "+string(rune(i))+" to be a string but got something else.",
						)
						return types.ObjectNull(map[string]attr.Type{}), diags
					}
					audiences = append(audiences, s)
				}

				audiencesList, listDiags := types.ListValueFrom(ctx, types.StringType, audiences)
				diags.Append(listDiags...)

				configuration["audiences"] = audiencesList
			}

			obj, objDiags := types.ObjectValue(map[string]attr.Type{
				"gateway_id": types.StringType,

				"auth_method": types.StringType,
				"api_config": types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"cluster_url":   types.StringType,
						"cluster_token": types.StringType,
						"enable_ssl":    types.BoolType,
						"ca":            types.StringType,
					},
				},

				"credential_type": types.StringType,
				"static_config": types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"service_account_name": types.StringType,
						"namespace":            types.StringType,
					},
				},
				"dynamic_config": types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"allowed_namespaces": types.StringType,
						"role":               types.StringType,
						"role_type":          types.StringType,
					},
				},

				"audiences": types.ListType{ElemType: types.StringType},
			}, configuration)
			if objDiags.HasError() {
				diags.Append(objDiags...)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}
			return obj, diags
		},
	}
}
