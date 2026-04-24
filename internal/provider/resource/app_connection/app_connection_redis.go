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

// AppConnectionRedisCredentialsModel describes the data source data model.
type AppConnectionRedisCredentialsModel struct {
	Host                  types.String `tfsdk:"host"`
	Port                  types.Int32  `tfsdk:"port"`
	Database              types.Int32  `tfsdk:"database"`
	Username              types.String `tfsdk:"username"`
	Password              types.String `tfsdk:"password"`
	SslEnabled            types.Bool   `tfsdk:"ssl_enabled"`
	SslRejectUnauthorized types.Bool   `tfsdk:"ssl_reject_unauthorized"`
	SslCertificate        types.String `tfsdk:"ssl_certificate"`
}

const AppConnectionRedisAuthMethodUsernameAndPassword = "username-and-password"
const AppConnectionRedisAuthMethodPassword = "password"

func NewAppConnectionRedisResource() resource.Resource {
	return &AppConnectionBaseResource{
		App:               infisical.AppConnectionAppRedis,
		AppConnectionName: "Redis",
		ResourceTypeName:  "_app_connection_redis",
		AllowedMethods:    []string{AppConnectionRedisAuthMethodUsernameAndPassword, AppConnectionRedisAuthMethodPassword},
		CredentialsAttributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Required:    true,
				Description: "The hostname of the Redis server.",
			},
			"port": schema.Int32Attribute{
				Optional:    true,
				Computed:    true,
				Description: "The port number of the Redis server.",
				Default:     int32default.StaticInt32(6379),
			},
			"database": schema.Int32Attribute{
				Optional:    true,
				Computed:    true,
				Description: "The Redis database number (0-15).",
				Default:     int32default.StaticInt32(0),
			},
			"username": schema.StringAttribute{
				Optional:    true,
				Description: "The username for Redis authentication (Redis 6+ ACL).",
			},
			"password": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "The password for Redis authentication.",
			},
			"ssl_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether or not to use SSL when connecting to Redis.",
				Default:     booldefault.StaticBool(false),
			},
			"ssl_reject_unauthorized": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether or not to reject unauthorized SSL certificates when connecting to Redis.",
				Default:     booldefault.StaticBool(true),
			},
			"ssl_certificate": schema.StringAttribute{
				Optional:    true,
				Description: "The SSL certificate to use for connection.",
			},
		},
		ReadCredentialsForCreateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			credentialsConfig := make(map[string]any)

			var credentials AppConnectionRedisCredentialsModel
			diags := plan.Credentials.As(ctx, &credentials, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != AppConnectionRedisAuthMethodUsernameAndPassword && plan.Method.ValueString() != AppConnectionRedisAuthMethodPassword {
				diags.AddError(
					"Unable to create Redis app connection",
					"Invalid method. Only username-and-password or password method is supported",
				)
				return nil, diags
			}

			credentialsConfig["host"] = credentials.Host.ValueString()
			credentialsConfig["port"] = credentials.Port.ValueInt32()
			credentialsConfig["database"] = credentials.Database.ValueInt32()
			credentialsConfig["username"] = credentials.Username.ValueString()
			credentialsConfig["password"] = credentials.Password.ValueString()
			credentialsConfig["sslEnabled"] = credentials.SslEnabled.ValueBool()
			credentialsConfig["sslRejectUnauthorized"] = credentials.SslRejectUnauthorized.ValueBool()
			credentialsConfig["sslCertificate"] = credentials.SslCertificate.ValueString()

			return credentialsConfig, diags
		},
		ReadCredentialsForUpdateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel, state AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			credentialsConfig := make(map[string]any)

			var credentialsFromPlan AppConnectionRedisCredentialsModel
			diags := plan.Credentials.As(ctx, &credentialsFromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var credentialsFromState AppConnectionRedisCredentialsModel
			diags = state.Credentials.As(ctx, &credentialsFromState, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != AppConnectionRedisAuthMethodUsernameAndPassword && plan.Method.ValueString() != AppConnectionRedisAuthMethodPassword {
				diags.AddError(
					"Unable to update Redis app connection",
					"Invalid method. Only username-and-password or password method is supported",
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
				credentialsConfig["database"] = database.ValueInt32()
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
				"database":                types.Int32Null(),
				"username":                types.StringNull(),
				"password":                types.StringNull(),
				"ssl_enabled":             types.BoolNull(),
				"ssl_reject_unauthorized": types.BoolNull(),
				"ssl_certificate":         types.StringNull(),
			}

			credentialsObj, diags := types.ObjectValue(
				map[string]attr.Type{
					"host":                    types.StringType,
					"port":                    types.Int32Type,
					"database":                types.Int32Type,
					"username":                types.StringType,
					"password":                types.StringType,
					"ssl_enabled":             types.BoolType,
					"ssl_reject_unauthorized": types.BoolType,
					"ssl_certificate":         types.StringType,
				},
				credentialsConfig,
			)

			if diags.HasError() {
				return diags
			}

			state.Credentials = credentialsObj
			return nil
		},
	}
}
