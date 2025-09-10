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

type DynamicSecretMongoDbConfigurationModel struct {
	Host     types.String `tfsdk:"host"`
	Port     types.Int64  `tfsdk:"port"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	Database types.String `tfsdk:"database"`
	Ca       types.String `tfsdk:"ca"`
	Roles    types.List   `tfsdk:"roles"`
}

func NewDynamicSecretMongoDbResource() resource.Resource {
	return &DynamicSecretBaseResource{
		Provider:          infisical.DynamicSecretProviderMongoDb,
		ResourceTypeName:  "_dynamic_secret_mongo_db",
		DynamicSecretName: "MongoDB",
		ConfigurationAttributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Required:    true,
				Description: "The host of the database server.",
			},
			"port": schema.Int64Attribute{
				Optional:    true,
				Description: "The port of the database server.",
			},
			"username": schema.StringAttribute{
				Required:    true,
				Description: "The username to use to connect to the database.",
			},
			"password": schema.StringAttribute{
				Required:    true,
				Description: "The password to use to connect to the database.",
				Sensitive:   true,
			},
			"database": schema.StringAttribute{
				Required:    true,
				Description: "The name of the database to use.",
			},
			"ca": schema.StringAttribute{
				Optional:    true,
				Description: "The CA certificate to use to connect to the database.",
			},
			"roles": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "A list of role names to assign to the user. The role names can either be built-in or custom.",
			},
		},

		ReadConfigurationFromPlan: func(ctx context.Context, plan DynamicSecretBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			configurationMap := make(map[string]interface{})
			var configuration DynamicSecretMongoDbConfigurationModel

			diags := plan.Configuration.As(ctx, &configuration, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			configurationMap["host"] = configuration.Host.ValueString()
			configurationMap["port"] = configuration.Port.ValueInt64()
			configurationMap["username"] = configuration.Username.ValueString()
			configurationMap["password"] = configuration.Password.ValueString()
			configurationMap["database"] = configuration.Database.ValueString()
			configurationMap["ca"] = configuration.Ca.ValueString()

			var roles []string
			diags.Append(configuration.Roles.ElementsAs(ctx, &roles, false)...)
			if diags.HasError() {
				return nil, diags
			}
			configurationMap["roles"] = roles

			return configurationMap, diags
		},

		ReadConfigurationFromApi: func(ctx context.Context, dynamicSecret infisical.DynamicSecret, configState types.Object) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

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

			databaseVal, ok := dynamicSecret.Inputs["database"].(string)
			if !ok {
				diags.AddError(
					"Invalid database type",
					"Expected 'database' to be a string but got something else",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			caVal, ok := dynamicSecret.Inputs["ca"].(string)
			if !ok {
				caVal = ""
			}

			caValue := types.StringNull()
			if caVal != "" {
				caValue = types.StringValue(caVal)
			}

			rolesVal, ok := dynamicSecret.Inputs["roles"].([]interface{})
			if !ok {
				diags.AddError(
					"Invalid roles type",
					"Expected 'roles' to be an array but got something else",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			roles := []attr.Value{}
			for _, role := range rolesVal {
				roleStr, ok := role.(string)
				if !ok {
					diags.AddError(
						"Invalid role type in roles list",
						"Expected 'role' to be a string but got something else",
					)
					return types.ObjectNull(map[string]attr.Type{}), diags
				}
				roles = append(roles, types.StringValue(roleStr))
			}

			rolesList, rolesDiags := types.ListValue(types.StringType, roles)
			if rolesDiags.HasError() {
				diags.Append(rolesDiags...)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			configuration := map[string]attr.Value{
				"host":     types.StringValue(hostVal),
				"port":     types.Int64Value(portVal),
				"username": types.StringValue(usernameVal),
				"password": types.StringValue(passwordVal),
				"database": types.StringValue(databaseVal),
				"ca":       caValue,
				"roles":    rolesList,
			}

			obj, objDiags := types.ObjectValue(map[string]attr.Type{
				"host":     types.StringType,
				"port":     types.Int64Type,
				"username": types.StringType,
				"password": types.StringType,
				"database": types.StringType,
				"ca":       types.StringType,
				"roles":    types.ListType{ElemType: types.StringType},
			}, configuration)
			if objDiags.HasError() {
				diags.Append(objDiags...)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}
			return obj, diags
		},
	}
}
