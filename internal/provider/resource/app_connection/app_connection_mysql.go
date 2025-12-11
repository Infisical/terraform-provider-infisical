package resource

import (
	"context"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32default"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// AppConnectionMySqlCredentialsModel describes the data source data model.
type AppConnectionMySqlCredentialsModel struct {
	Host                  types.String `tfsdk:"host"`
	Port                  types.Int32  `tfsdk:"port"`
	Database              types.String `tfsdk:"database"`
	Username              types.String `tfsdk:"username"`
	Password              types.String `tfsdk:"password"`
	SslEnabled            types.Bool   `tfsdk:"ssl_enabled"`
	SslRejectUnauthorized types.Bool   `tfsdk:"ssl_reject_unauthorized"`
	SslCertificate        types.String `tfsdk:"ssl_certificate"`
}

const AppConnectionMySqlAuthMethodUsernameAndPassword = "username-and-password"

func NewAppConnectionMySqlResource() resource.Resource {
	return &AppConnectionBaseResource{
		App:               infisical.AppConnectionAppMySql,
		AppConnectionName: "MySQL",
		ResourceTypeName:  "_app_connection_mysql",
		AllowedMethods:    []string{AppConnectionMySqlAuthMethodUsernameAndPassword},
		CredentialsAttributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Required:    true,
				Description: "The hostname of the database server.",
			},
			"port": schema.Int32Attribute{
				Optional:    true,
				Computed:    true,
				Description: "The port number of the database.",
				Default:     int32default.StaticInt32(3306),
			},
			"database": schema.StringAttribute{
				Required:    true,
				Description: "The name of the database to connect to.",
			},
			"username": schema.StringAttribute{
				Required:    true,
				Description: "The username to connect to the database with.",
			},
			"password": schema.StringAttribute{
				Required:    true,
				Description: "The password to connect to the database with.",
				Sensitive:   true,
			},
			"ssl_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether or not to use SSL when connecting to the database.",
				Default:     booldefault.StaticBool(true),
			},
			"ssl_reject_unauthorized": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether or not to reject unauthorized SSL certificates.",
				Default:     booldefault.StaticBool(true),
			},
			"ssl_certificate": schema.StringAttribute{
				Optional:    true,
				Description: "The SSL certificate to use for connection.",
			},
		},
		ReadCredentialsForCreateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			credentialsConfig := make(map[string]any)

			var credentials AppConnectionMySqlCredentialsModel
			diags := plan.Credentials.As(ctx, &credentials, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != AppConnectionMySqlAuthMethodUsernameAndPassword {
				diags.AddError(
					"Unable to create MySQL app connection",
					"Invalid method. Only username-and-password method is supported",
				)
				return nil, diags
			}

			credentialsConfig["host"] = credentials.Host.ValueString()
			credentialsConfig["port"] = credentials.Port.ValueInt32()
			credentialsConfig["database"] = credentials.Database.ValueString()
			credentialsConfig["username"] = credentials.Username.ValueString()
			credentialsConfig["password"] = credentials.Password.ValueString()
			credentialsConfig["sslEnabled"] = credentials.SslEnabled.ValueBool()
			credentialsConfig["sslRejectUnauthorized"] = credentials.SslRejectUnauthorized.ValueBool()
			credentialsConfig["sslCertificate"] = credentials.SslCertificate.ValueString()

			return credentialsConfig, diags
		},
		ReadCredentialsForUpdateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel, state AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			credentialsConfig := make(map[string]any)

			var credentialsFromPlan AppConnectionMySqlCredentialsModel
			diags := plan.Credentials.As(ctx, &credentialsFromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var credentialsFromState AppConnectionMySqlCredentialsModel
			diags = state.Credentials.As(ctx, &credentialsFromState, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != AppConnectionMySqlAuthMethodUsernameAndPassword {
				diags.AddError(
					"Unable to update MySQL app connection",
					"Invalid method. Only username-and-password method is supported",
				)
				return nil, diags
			}

			host := credentialsFromPlan.Host
			if credentialsFromPlan.Host.IsUnknown() {
				host = credentialsFromState.Host
			}
			if !host.IsNull() {
				credentialsConfig["host"] = host.ValueString()
			}

			port := credentialsFromPlan.Port
			if credentialsFromPlan.Port.IsUnknown() {
				port = credentialsFromState.Port
			}
			if !port.IsNull() {
				credentialsConfig["port"] = port.ValueInt32()
			}

			database := credentialsFromPlan.Database
			if credentialsFromPlan.Database.IsUnknown() {
				database = credentialsFromState.Database
			}
			if !database.IsNull() {
				credentialsConfig["database"] = database.ValueString()
			}

			username := credentialsFromPlan.Username
			if credentialsFromPlan.Username.IsUnknown() {
				username = credentialsFromState.Username
			}
			if !username.IsNull() {
				credentialsConfig["username"] = username.ValueString()
			}

			password := credentialsFromPlan.Password
			if credentialsFromPlan.Password.IsUnknown() {
				password = credentialsFromState.Password
			}
			if !password.IsNull() {
				credentialsConfig["password"] = password.ValueString()
			}

			sslEnabled := credentialsFromPlan.SslEnabled
			if credentialsFromPlan.SslEnabled.IsUnknown() {
				sslEnabled = credentialsFromState.SslEnabled
			}
			if !sslEnabled.IsNull() {
				credentialsConfig["sslEnabled"] = sslEnabled.ValueBool()
			}

			sslRejectUnauthorized := credentialsFromPlan.SslRejectUnauthorized
			if credentialsFromPlan.SslRejectUnauthorized.IsUnknown() {
				sslRejectUnauthorized = credentialsFromState.SslRejectUnauthorized
			}
			if !sslRejectUnauthorized.IsNull() {
				credentialsConfig["sslRejectUnauthorized"] = sslRejectUnauthorized.ValueBool()
			}

			sslCertificate := credentialsFromPlan.SslCertificate
			if credentialsFromPlan.SslCertificate.IsUnknown() {
				sslCertificate = credentialsFromState.SslCertificate
			}
			if !sslCertificate.IsNull() {
				credentialsConfig["sslCertificate"] = sslCertificate.ValueString()
			}

			return credentialsConfig, diags
		},
		OverwriteCredentialsFields: func(state *AppConnectionBaseResourceModel) diag.Diagnostics {
			credentialsConfig := map[string]attr.Value{
				"host":                    types.StringNull(),
				"port":                    types.Int32Null(),
				"database":                types.StringNull(),
				"username":                types.StringNull(),
				"password":                types.StringNull(),
				"ssl_enabled":             types.BoolNull(),
				"ssl_reject_unauthorized": types.BoolNull(),
				"ssl_certificate":         types.StringNull(),
			}

			var diags diag.Diagnostics
			state.Credentials, diags = types.ObjectValue(map[string]attr.Type{
				"host":                    types.StringType,
				"port":                    types.Int32Type,
				"database":                types.StringType,
				"username":                types.StringType,
				"password":                types.StringType,
				"ssl_enabled":             types.BoolType,
				"ssl_reject_unauthorized": types.BoolType,
				"ssl_certificate":         types.StringType,
			}, credentialsConfig)

			return diags
		},
	}
}
