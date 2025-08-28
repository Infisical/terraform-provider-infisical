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

type SecretRotationLdapPasswordParametersModel struct {
	Dn                      types.String `tfsdk:"dn"`
	PasswordRequirements    types.Object `tfsdk:"password_requirements"`
	RotationMethod          types.String `tfsdk:"rotation_method"`
	TargetPrincipalPassword types.String `tfsdk:"target_principal_password"`
}

type PasswordRequirementsModel struct {
	Length         types.Int64  `tfsdk:"length"`
	Required       types.Object `tfsdk:"required"`
	AllowedSymbols types.String `tfsdk:"allowed_symbols"`
}

type RequiredCharactersModel struct {
	Digits    types.Int64 `tfsdk:"digits"`
	Lowercase types.Int64 `tfsdk:"lowercase"`
	Uppercase types.Int64 `tfsdk:"uppercase"`
	Symbols   types.Int64 `tfsdk:"symbols"`
}

type SecretRotationLdapPasswordSecretsMappingModel struct {
	Dn       types.String `tfsdk:"dn"`
	Password types.String `tfsdk:"password"`
}

func NewSecretRotationLdapPasswordResource() resource.Resource {
	return &SecretRotationBaseResource{
		Provider:           infisical.SecretRotationProviderLdapPassword,
		SecretRotationName: "LDAP Password",
		ResourceTypeName:   "_secret_rotation_ldap_password",
		AppConnection:      infisical.AppConnectionAppLdap,
		ParametersAttributes: map[string]schema.Attribute{
			"dn": schema.StringAttribute{
				Required:    true,
				Description: "The Distinguished Name (DN) of the LDAP entry to rotate the password for.",
			},
			"password_requirements": schema.SingleNestedAttribute{
				Required:    true,
				Description: "Password generation requirements.",
				Attributes: map[string]schema.Attribute{
					"length": schema.Int64Attribute{
						Required:    true,
						Description: "The length of the generated password.",
					},
					"required": schema.SingleNestedAttribute{
						Required:    true,
						Description: "Required character types in the generated password.",
						Attributes: map[string]schema.Attribute{
							"digits": schema.Int64Attribute{
								Required:    true,
								Description: "Minimum number of digits required in the password.",
							},
							"lowercase": schema.Int64Attribute{
								Required:    true,
								Description: "Minimum number of lowercase letters required in the password.",
							},
							"uppercase": schema.Int64Attribute{
								Required:    true,
								Description: "Minimum number of uppercase letters required in the password.",
							},
							"symbols": schema.Int64Attribute{
								Required:    true,
								Description: "Minimum number of symbols required in the password.",
							},
						},
					},
					"allowed_symbols": schema.StringAttribute{
						Optional:    true,
						Description: "String of allowed symbols for password generation.",
					},
				},
			},
			"rotation_method": schema.StringAttribute{
				Optional:    true,
				Description: "The method to use for rotating the password. Supported options: connection-principal and target-principal (default: connection-principal)",
			},
			"target_principal_password": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "The temporary password for the target principal. Required when rotation_method is 'target-principal'.",
			},
		},
		SecretsMappingAttributes: map[string]schema.Attribute{
			"dn": schema.StringAttribute{
				Required:    true,
				Description: "The name of the secret that the Distinguished Name will be mapped to.",
			},
			"password": schema.StringAttribute{
				Required:    true,
				Description: "The name of the secret that the generated password will be mapped to.",
			},
		},

		ReadParametersFromPlan: func(ctx context.Context, plan SecretRotationBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			parametersMap := make(map[string]interface{})
			var parameters SecretRotationLdapPasswordParametersModel

			diags := plan.Parameters.As(ctx, &parameters, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			parametersMap["dn"] = parameters.Dn.ValueString()

			// Handle password requirements
			if !parameters.PasswordRequirements.IsNull() {
				var passwordReqs PasswordRequirementsModel
				diags = parameters.PasswordRequirements.As(ctx, &passwordReqs, basetypes.ObjectAsOptions{})
				if diags.HasError() {
					return nil, diags
				}

				passwordReqsMap := make(map[string]interface{})
				passwordReqsMap["length"] = passwordReqs.Length.ValueInt64()

				// Handle required characters
				if !passwordReqs.Required.IsNull() {
					var requiredChars RequiredCharactersModel
					diags = passwordReqs.Required.As(ctx, &requiredChars, basetypes.ObjectAsOptions{})
					if diags.HasError() {
						return nil, diags
					}

					requiredMap := make(map[string]interface{})
					requiredMap["digits"] = requiredChars.Digits.ValueInt64()
					requiredMap["lowercase"] = requiredChars.Lowercase.ValueInt64()
					requiredMap["uppercase"] = requiredChars.Uppercase.ValueInt64()
					requiredMap["symbols"] = requiredChars.Symbols.ValueInt64()
					passwordReqsMap["required"] = requiredMap
				}

				if !passwordReqs.AllowedSymbols.IsNull() {
					passwordReqsMap["allowedSymbols"] = passwordReqs.AllowedSymbols.ValueString()
				}

				parametersMap["passwordRequirements"] = passwordReqsMap
			}

			if !parameters.RotationMethod.IsNull() {
				parametersMap["rotationMethod"] = parameters.RotationMethod.ValueString()
			}

			if parameters.RotationMethod.String() == "target-principal" && parameters.TargetPrincipalPassword.IsNull() {
				diags.AddError("Plan Error", "Expected 'target_principal_password' (string) but got wrong type or missing")
			}

			if diags.HasError() {
				return nil, diags
			}

			if !parameters.TargetPrincipalPassword.IsNull() {
				temporaryParams := make(map[string]any)
				temporaryParams["targetPrincipalPassword"] = parameters.TargetPrincipalPassword.ValueString()
				parametersMap["temporaryParameters"] = temporaryParams
			}

			return parametersMap, diags
		},

		ReadParametersFromApi: func(ctx context.Context, secretRotation infisicalclient.SecretRotation) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics
			parameters := make(map[string]attr.Value)
			parametersSchema := map[string]attr.Type{
				"dn": types.StringType,
				"password_requirements": types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"length": types.Int64Type,
						"required": types.ObjectType{
							AttrTypes: map[string]attr.Type{
								"digits":    types.Int64Type,
								"lowercase": types.Int64Type,
								"uppercase": types.Int64Type,
								"symbols":   types.Int64Type,
							},
						},
						"allowed_symbols": types.StringType,
					},
				},
				"rotation_method":           types.StringType,
				"target_principal_password": types.StringType,
			}

			// Extract DN
			dnVal, ok := secretRotation.Parameters["dn"].(string)
			if !ok {
				diags.AddError("API Reading Error", "Expected 'dn' (string) but got wrong type or missing")
				return types.ObjectNull(parametersSchema), diags
			}
			parameters["dn"] = types.StringValue(dnVal)

			// Extract password requirements
			if passwordReqsVal, ok := secretRotation.Parameters["passwordRequirements"].(map[string]interface{}); ok {
				passwordReqsAttrs := make(map[string]attr.Value)

				// Extract length
				if lengthVal, ok := passwordReqsVal["length"].(float64); ok {
					passwordReqsAttrs["length"] = types.Int64Value(int64(lengthVal))
				} else {
					passwordReqsAttrs["length"] = types.Int64Value(0)
				}

				// Extract required characters
				if requiredVal, ok := passwordReqsVal["required"].(map[string]interface{}); ok {
					requiredAttrs := make(map[string]attr.Value)

					if digitsVal, ok := requiredVal["digits"].(float64); ok {
						requiredAttrs["digits"] = types.Int64Value(int64(digitsVal))
					} else {
						requiredAttrs["digits"] = types.Int64Value(0)
					}

					if lowercaseVal, ok := requiredVal["lowercase"].(float64); ok {
						requiredAttrs["lowercase"] = types.Int64Value(int64(lowercaseVal))
					} else {
						requiredAttrs["lowercase"] = types.Int64Value(0)
					}

					if uppercaseVal, ok := requiredVal["uppercase"].(float64); ok {
						requiredAttrs["uppercase"] = types.Int64Value(int64(uppercaseVal))
					} else {
						requiredAttrs["uppercase"] = types.Int64Value(0)
					}

					if symbolsVal, ok := requiredVal["symbols"].(float64); ok {
						requiredAttrs["symbols"] = types.Int64Value(int64(symbolsVal))
					} else {
						requiredAttrs["symbols"] = types.Int64Value(0)
					}

					requiredObj, objDiags := types.ObjectValue(map[string]attr.Type{
						"digits":    types.Int64Type,
						"lowercase": types.Int64Type,
						"uppercase": types.Int64Type,
						"symbols":   types.Int64Type,
					}, requiredAttrs)
					diags.Append(objDiags...)
					passwordReqsAttrs["required"] = requiredObj
				} else {
					passwordReqsAttrs["required"] = types.ObjectNull(map[string]attr.Type{
						"digits":    types.Int64Type,
						"lowercase": types.Int64Type,
						"uppercase": types.Int64Type,
						"symbols":   types.Int64Type,
					})
				}

				// Extract allowed symbols
				if allowedSymbolsVal, ok := passwordReqsVal["allowedSymbols"].(string); ok {
					passwordReqsAttrs["allowed_symbols"] = types.StringValue(allowedSymbolsVal)
				} else {
					passwordReqsAttrs["allowed_symbols"] = types.StringNull()
				}

				passwordReqsObj, objDiags := types.ObjectValue(map[string]attr.Type{
					"length": types.Int64Type,
					"required": types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"digits":    types.Int64Type,
							"lowercase": types.Int64Type,
							"uppercase": types.Int64Type,
							"symbols":   types.Int64Type,
						},
					},
					"allowed_symbols": types.StringType,
				}, passwordReqsAttrs)
				diags.Append(objDiags...)
				parameters["password_requirements"] = passwordReqsObj
			} else {
				diags.AddError("API Reading Error", "Expected 'passwordRequirements' (object) but got wrong type or missing")
				return types.ObjectNull(parametersSchema), diags
			}

			// Extract rotation method if present
			if rotationMethodVal, ok := secretRotation.Parameters["rotationMethod"].(string); ok {
				parameters["rotation_method"] = types.StringValue(rotationMethodVal)
			} else {
				parameters["rotation_method"] = types.StringNull()
			}

			// Extract target principal password from temporaryParameters if present
			if temporaryParams, ok := secretRotation.Parameters["temporaryParameters"].(map[string]any); ok {
				if targetPassword, ok := temporaryParams["targetPrincipalPassword"].(string); ok {
					parameters["target_principal_password"] = types.StringValue(targetPassword)
				} else {
					parameters["target_principal_password"] = types.StringNull()
				}
			} else {
				parameters["target_principal_password"] = types.StringNull()
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
			var secretsMapping SecretRotationLdapPasswordSecretsMappingModel

			diags := plan.SecretsMapping.As(ctx, &secretsMapping, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			secretsMappingMap["dn"] = secretsMapping.Dn.ValueString()
			secretsMappingMap["password"] = secretsMapping.Password.ValueString()

			return secretsMappingMap, diags
		},

		ReadSecretsMappingFromApi: func(ctx context.Context, secretRotation infisicalclient.SecretRotation) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics
			secretsMapping := make(map[string]attr.Value)
			secretsMappingSchema := map[string]attr.Type{
				"dn":       types.StringType,
				"password": types.StringType,
			}

			dnVal, ok := secretRotation.SecretsMapping["dn"].(string)
			if !ok {
				diags.AddError("API Reading Error", "Expected 'dn' (string) but got wrong type or missing")
				return types.ObjectNull(secretsMappingSchema), diags
			}
			secretsMapping["dn"] = types.StringValue(dnVal)

			passwordVal, ok := secretRotation.SecretsMapping["password"].(string)
			if !ok {
				diags.AddError("API Reading Error", "Expected 'password' (string) but got wrong type or missing")
				return types.ObjectNull(secretsMappingSchema), diags
			}
			secretsMapping["password"] = types.StringValue(passwordVal)

			obj, objDiags := types.ObjectValue(secretsMappingSchema, secretsMapping)
			diags.Append(objDiags...)
			if diags.HasError() {
				return types.ObjectNull(secretsMappingSchema), diags
			}

			return obj, diags
		},
	}
}
