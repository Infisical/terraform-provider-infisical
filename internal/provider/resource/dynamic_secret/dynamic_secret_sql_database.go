package resource

import (
	"context"
	infisical "terraform-provider-infisical/internal/client"
	pkg "terraform-provider-infisical/internal/pkg/modifiers"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type PasswordRequirementsRequiredModel struct {
	Lowercase types.Int64 `tfsdk:"lowercase"`
	Uppercase types.Int64 `tfsdk:"uppercase"`
	Digits    types.Int64 `tfsdk:"digits"`
	Symbols   types.Int64 `tfsdk:"symbols"`
}

type PasswordRequirementsModel struct {
	Length         types.Int64                       `tfsdk:"length"`
	Required       PasswordRequirementsRequiredModel `tfsdk:"required"`
	AllowedSymbols types.String                      `tfsdk:"allowed_symbols"`
}

// AppConnectionAwsCredentialsModel describes the data source data model.
type DynamicSecretSqlDatabaseConfigurationModel struct {
	Client               types.String               `tfsdk:"client"`
	Host                 types.String               `tfsdk:"host"`
	Port                 types.Int64                `tfsdk:"port"`
	Database             types.String               `tfsdk:"database"`
	Username             types.String               `tfsdk:"username"`
	Password             types.String               `tfsdk:"password"`
	PasswordWO           types.String               `tfsdk:"password_wo"`
	PasswordWOVersion    types.Int64                `tfsdk:"password_wo_version"`
	CreationStatement    types.String               `tfsdk:"creation_statement"`
	RevocationStatement  types.String               `tfsdk:"revocation_statement"`
	RenewStatement       types.String               `tfsdk:"renew_statement"`
	CertificateAuthority types.String               `tfsdk:"ca"`
	GatewayId            types.String               `tfsdk:"gateway_id"`
	PasswordRequirements *PasswordRequirementsModel `tfsdk:"password_requirements"`
}

func NewDynamicSecretSqlDatabaseResource() resource.Resource {
	return &DynamicSecretBaseResource{
		Provider:          infisical.DynamicSecretProviderSQLDatabase,
		ResourceTypeName:  "_dynamic_secret_sql_database",
		DynamicSecretName: "SQL Database",
		ConfigurationAttributes: map[string]schema.Attribute{
			"client": schema.StringAttribute{
				Required:    true,
				Description: "The database client to use. Currently supported values are postgres, mysql2, oracledb, mssql, sap-ase, and vertica.",
			},
			"host": schema.StringAttribute{
				Required:    true,
				Description: "The host of the database server.",
			},
			"port": schema.Int64Attribute{
				Required:    true,
				Description: "The port of the database server.",
			},
			"database": schema.StringAttribute{
				Required:    true,
				Description: "The name of the database to use.",
			},
			"username": schema.StringAttribute{
				Required:    true,
				Description: "The username to use to connect to the database.",
			},
			"password": schema.StringAttribute{
				Optional:    true,
				Description: "The password to use to connect to the database. This value is stored in the Terraform state; use password_wo to keep it out of state. Exactly one of password or password_wo must be set.",
				Sensitive:   true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(
						path.MatchRelative().AtParent().AtName("password_wo"),
					),
				},
			},
			"password_wo": schema.StringAttribute{
				Optional:    true,
				WriteOnly:   true,
				Description: "The password to use to connect to the database (write-only). This value is never stored in the Terraform state and can accept ephemeral values. Because it is not stored, changes to it are not detected; increment password_wo_version to push a new value. Requires Terraform 1.11+.",
				Sensitive:   true,
				Validators: []validator.String{
					stringvalidator.AlsoRequires(
						path.MatchRelative().AtParent().AtName("password_wo_version"),
					),
				},
			},
			"password_wo_version": schema.Int64Attribute{
				Optional:    true,
				Description: "The version of the password_wo value. Increment this to trigger an update of the password.",
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
					int64validator.AlsoRequires(
						path.MatchRelative().AtParent().AtName("password_wo"),
					),
				},
			},
			"creation_statement": schema.StringAttribute{
				Required:    true,
				Description: "The creation statement to use to create the dynamic secret lease.",
				PlanModifiers: []planmodifier.String{
					pkg.TrimEqualityModifier{},
				},
			},
			"revocation_statement": schema.StringAttribute{
				Required:    true,
				Description: "The revocation statement to use to revoke the dynamic secret lease.",
				PlanModifiers: []planmodifier.String{
					pkg.TrimEqualityModifier{},
				},
			},
			"renew_statement": schema.StringAttribute{
				Optional:    true,
				Description: "The renew statement to use to renew the dynamic secret lease.",
				PlanModifiers: []planmodifier.String{
					pkg.TrimEqualityModifier{},
				},
			},
			"password_requirements": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "The password requirements to use to create the dynamic secret lease.",
				Attributes: map[string]schema.Attribute{
					"length": schema.Int64Attribute{
						Required:    true,
						Description: "The length of the password to use to create the dynamic secret lease.",
					},
					"required": schema.SingleNestedAttribute{
						Required:    true,
						Description: "The required characters to use to create the dynamic secret lease.",
						Attributes: map[string]schema.Attribute{
							"lowercase": schema.Int64Attribute{
								Required:    true,
								Description: "The number of lowercase characters required in the password.",
							},
							"uppercase": schema.Int64Attribute{
								Required:    true,
								Description: "The number of uppercase characters required in the password.",
							},
							"digits": schema.Int64Attribute{
								Required:    true,
								Description: "The number of digits required in the password.",
							},
							"symbols": schema.Int64Attribute{
								Required:    true,
								Description: "The number of symbols required in the password.",
							},
						},
					},
					"allowed_symbols": schema.StringAttribute{
						Optional:    true,
						Description: "The symbols allowed in the password.",
					},
				},
			},
			"ca": schema.StringAttribute{
				Optional:    true,
				Description: "The CA certificate to use to connect to the database.",
			},
			"gateway_id": schema.StringAttribute{
				Optional:    true,
				Description: "The Gateway ID to use to connect to the database.",
			},
		},

		ReadConfigurationFromPlan: func(ctx context.Context, plan DynamicSecretBaseResourceModel, config DynamicSecretBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			configurationMap := make(map[string]interface{})
			var configuration DynamicSecretSqlDatabaseConfigurationModel

			diags := plan.Configuration.As(ctx, &configuration, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			// Write-only values are stripped from the plan, so read them from config
			var rawConfiguration DynamicSecretSqlDatabaseConfigurationModel
			diags.Append(config.Configuration.As(ctx, &rawConfiguration, basetypes.ObjectAsOptions{})...)
			if diags.HasError() {
				return nil, diags
			}

			password := configuration.Password.ValueString()
			if !rawConfiguration.PasswordWO.IsNull() {
				password = rawConfiguration.PasswordWO.ValueString()
			}

			configurationMap["client"] = configuration.Client.ValueString()
			configurationMap["host"] = configuration.Host.ValueString()
			configurationMap["port"] = configuration.Port.ValueInt64()
			configurationMap["database"] = configuration.Database.ValueString()
			configurationMap["username"] = configuration.Username.ValueString()
			configurationMap["password"] = password
			configurationMap["creationStatement"] = configuration.CreationStatement.ValueString()
			configurationMap["revocationStatement"] = configuration.RevocationStatement.ValueString()
			configurationMap["renewStatement"] = configuration.RenewStatement.ValueString()
			configurationMap["ca"] = configuration.CertificateAuthority.ValueString()
			configurationMap["gatewayId"] = configuration.GatewayId.ValueString()

			// Only include password requirements if defined
			if configuration.PasswordRequirements != nil {
				passwordReqMap := make(map[string]interface{})
				passwordReqMap["length"] = configuration.PasswordRequirements.Length.ValueInt64()
				passwordReqMap["allowedSymbols"] = configuration.PasswordRequirements.AllowedSymbols.ValueString()

				requiredMap := make(map[string]interface{})
				requiredMap["lowercase"] = configuration.PasswordRequirements.Required.Lowercase.ValueInt64()
				requiredMap["uppercase"] = configuration.PasswordRequirements.Required.Uppercase.ValueInt64()
				requiredMap["digits"] = configuration.PasswordRequirements.Required.Digits.ValueInt64()
				requiredMap["symbols"] = configuration.PasswordRequirements.Required.Symbols.ValueInt64()

				passwordReqMap["required"] = requiredMap

				configurationMap["passwordRequirements"] = passwordReqMap
			}

			return configurationMap, diags
		},

		ReadConfigurationFromApi: func(ctx context.Context, dynamicSecret infisical.DynamicSecret, configState types.Object) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			// Extract existing config state to preserve user formatting when values are equivalent after trimming
			var existingConfig DynamicSecretSqlDatabaseConfigurationModel
			if !configState.IsNull() {
				configDiags := configState.As(ctx, &existingConfig, basetypes.ObjectAsOptions{})
				if configDiags.HasError() {
					diags.Append(configDiags...)
				}
			}

			clientVal, ok := dynamicSecret.Inputs["client"].(string)
			if !ok {
				diags.AddError(
					"Invalid client type",
					"Expected 'client' to be a string but got something else",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			hostVal, ok := dynamicSecret.Inputs["host"].(string)
			if !ok {
				diags.AddError(
					"Invalid host type",
					"Expected 'host' to be a string but got something else",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			portFloat, ok := dynamicSecret.Inputs["port"].(float64)
			if !ok {
				diags.AddError(
					"Invalid port type",
					"Expected 'port' to be a float64 but got something else",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}
			portVal := int64(portFloat)

			databaseVal, ok := dynamicSecret.Inputs["database"].(string)
			if !ok {
				diags.AddError(
					"Invalid database type",
					"Expected 'database' to be a string but got something else",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			usernameVal, ok := dynamicSecret.Inputs["username"].(string)
			if !ok {
				diags.AddError(
					"Invalid username type",
					"Expected 'username' to be a string but got something else",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			passwordVal, ok := dynamicSecret.Inputs["password"].(string)
			if !ok {
				diags.AddError(
					"Invalid password type",
					"Expected 'password' to be a string but got something else",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			creationStatementVal, ok := dynamicSecret.Inputs["creationStatement"].(string)
			if !ok {
				diags.AddError(
					"Invalid creation statement type",
					"Expected 'creationStatement' to be a string but got something else",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			revocationStatementVal, ok := dynamicSecret.Inputs["revocationStatement"].(string)
			if !ok {
				diags.AddError(
					"Invalid revocation statement type",
					"Expected 'revocationStatement' to be a string but got something else",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			renewStatementVal, ok := dynamicSecret.Inputs["renewStatement"].(string)
			if !ok {
				renewStatementVal = ""
			}

			caVal, ok := dynamicSecret.Inputs["ca"].(string)
			if !ok {
				caVal = ""
			}

			gatewayIdVal, ok := dynamicSecret.Inputs["gatewayId"].(string)
			if !ok {
				gatewayIdVal = ""
			}

			gatewayIdValue := types.StringNull()
			if gatewayIdVal != "" {
				gatewayIdValue = types.StringValue(gatewayIdVal)
			}

			caValue := types.StringNull()
			if caVal != "" {
				caValue = types.StringValue(caVal)
			}

			creationStatementFinal := infisicaltf.PreserveStringIfTrimmedEqual(creationStatementVal, existingConfig.CreationStatement)
			revocationStatementFinal := infisicaltf.PreserveStringIfTrimmedEqual(revocationStatementVal, existingConfig.RevocationStatement)
			renewStatementFinal := infisicaltf.PreserveStringIfTrimmedEqual(renewStatementVal, existingConfig.RenewStatement)

			renewStatementValue := types.StringNull()
			if renewStatementFinal != "" {
				renewStatementValue = types.StringValue(renewStatementFinal)
			}

			// When password_wo is used, password is null in plan/state and must
			// stay null so the API's value is never written to the state.
			passwordValue := types.StringNull()
			if !existingConfig.Password.IsNull() {
				passwordValue = types.StringValue(passwordVal)
			}

			configuration := map[string]attr.Value{
				"client":               types.StringValue(clientVal),
				"host":                 types.StringValue(hostVal),
				"port":                 types.Int64Value(portVal),
				"database":             types.StringValue(databaseVal),
				"username":             types.StringValue(usernameVal),
				"password":             passwordValue,
				"password_wo":          types.StringNull(),
				"password_wo_version":  existingConfig.PasswordWOVersion,
				"creation_statement":   types.StringValue(creationStatementFinal),
				"revocation_statement": types.StringValue(revocationStatementFinal),
				"renew_statement":      renewStatementValue,
				"ca":                   caValue,
				"gateway_id":           gatewayIdValue,
			}

			// Handle password requirements if present
			if passwordReq, ok := dynamicSecret.Inputs["passwordRequirements"].(map[string]interface{}); ok {
				requiredMap := make(map[string]attr.Value)
				if required, ok := passwordReq["required"].(map[string]interface{}); ok {
					lowercase, _ := required["lowercase"].(float64)
					uppercase, _ := required["uppercase"].(float64)
					digits, _ := required["digits"].(float64)
					symbols, _ := required["symbols"].(float64)

					requiredMap["lowercase"] = types.Int64Value(int64(lowercase))
					requiredMap["uppercase"] = types.Int64Value(int64(uppercase))
					requiredMap["digits"] = types.Int64Value(int64(digits))
					requiredMap["symbols"] = types.Int64Value(int64(symbols))
				}

				length, _ := passwordReq["length"].(float64)
				allowedSymbols, _ := passwordReq["allowedSymbols"].(string)

				passwordReqMap := map[string]attr.Value{
					"length":          types.Int64Value(int64(length)),
					"allowed_symbols": types.StringValue(allowedSymbols),
				}

				requiredObj, requiredDiags := types.ObjectValue(map[string]attr.Type{
					"lowercase": types.Int64Type,
					"uppercase": types.Int64Type,
					"digits":    types.Int64Type,
					"symbols":   types.Int64Type,
				}, requiredMap)
				if requiredDiags.HasError() {
					diags.Append(requiredDiags...)
					return types.ObjectNull(map[string]attr.Type{}), diags
				}
				passwordReqMap["required"] = requiredObj

				passwordReqObj, passwordReqDiags := types.ObjectValue(map[string]attr.Type{
					"length":          types.Int64Type,
					"allowed_symbols": types.StringType,
					"required": types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"lowercase": types.Int64Type,
							"uppercase": types.Int64Type,
							"digits":    types.Int64Type,
							"symbols":   types.Int64Type,
						},
					},
				}, passwordReqMap)
				if passwordReqDiags.HasError() {
					diags.Append(passwordReqDiags...)
					return types.ObjectNull(map[string]attr.Type{}), diags
				}
				configuration["password_requirements"] = passwordReqObj
			} else {
				configuration["password_requirements"] = types.ObjectNull(map[string]attr.Type{
					"length":          types.Int64Type,
					"allowed_symbols": types.StringType,
					"required": types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"lowercase": types.Int64Type,
							"uppercase": types.Int64Type,
							"digits":    types.Int64Type,
							"symbols":   types.Int64Type,
						},
					},
				})
			}

			obj, objDiags := types.ObjectValue(map[string]attr.Type{
				"client":               types.StringType,
				"host":                 types.StringType,
				"port":                 types.Int64Type,
				"database":             types.StringType,
				"username":             types.StringType,
				"password":             types.StringType,
				"password_wo":          types.StringType,
				"password_wo_version":  types.Int64Type,
				"creation_statement":   types.StringType,
				"revocation_statement": types.StringType,
				"renew_statement":      types.StringType,
				"ca":                   types.StringType,
				"gateway_id":           types.StringType,
				"password_requirements": types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"length":          types.Int64Type,
						"allowed_symbols": types.StringType,
						"required": types.ObjectType{
							AttrTypes: map[string]attr.Type{
								"lowercase": types.Int64Type,
								"uppercase": types.Int64Type,
								"digits":    types.Int64Type,
								"symbols":   types.Int64Type,
							},
						},
					},
				},
			}, configuration)
			if objDiags.HasError() {
				diags.Append(objDiags...)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}
			return obj, diags
		},
	}
}
