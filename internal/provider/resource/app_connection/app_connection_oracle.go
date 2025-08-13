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

// AppConnectionOracleCredentialsModel describes the data source data model.
type AppConnectionOracleCredentialsModel struct {
	Host                  types.String `tfsdk:"host"`
	Port                  types.Int32  `tfsdk:"port"`
	ServiceName           types.String `tfsdk:"service_name"`
	Username              types.String `tfsdk:"username"`
	Password              types.String `tfsdk:"password"`
	SslEnabled            types.Bool   `tfsdk:"ssl_enabled"`
	SslRejectUnauthorized types.Bool   `tfsdk:"ssl_reject_unauthorized"`
	SslCertificate        types.String `tfsdk:"ssl_certificate"`
}

const AppConnectionOracleAuthMethodUsernameAndPassword = "username-and-password"

func NewAppConnectionOracleResource() resource.Resource {
	return &AppConnectionBaseResource{
		App:               infisical.AppConnectionAppOracle,
		AppConnectionName: "Oracle",
		ResourceTypeName:  "_app_connection_oracle",
		AllowedMethods:    []string{AppConnectionOracleAuthMethodUsernameAndPassword},
		CredentialsAttributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Required:    true,
				Description: "The hostname of the Oracle database server.",
			},
			"port": schema.Int32Attribute{
				Optional:    true,
				Computed:    true,
				Description: "The port number of the Oracle database.",
				Default:     int32default.StaticInt32(1521),
			},
			"service_name": schema.StringAttribute{
				Required:    true,
				Description: "The service name of the Oracle database.",
			},
			"username": schema.StringAttribute{
				Required:    true,
				Description: "The username to connect to the Oracle database with.",
			},
			"password": schema.StringAttribute{
				Required:    true,
				Description: "The password to connect to the Oracle database with.",
				Sensitive:   true,
			},
			"ssl_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether or not to use SSL when connecting to the Oracle database.",
				Default:     booldefault.StaticBool(false),
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

			var credentials AppConnectionOracleCredentialsModel
			diags := plan.Credentials.As(ctx, &credentials, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != AppConnectionOracleAuthMethodUsernameAndPassword {
				diags.AddError(
					"Unable to create Oracle app connection",
					"Invalid method. Only username-and-password method is supported",
				)
				return nil, diags
			}

			credentialsConfig["host"] = credentials.Host.ValueString()
			credentialsConfig["port"] = credentials.Port.ValueInt32()
			credentialsConfig["serviceName"] = credentials.ServiceName.ValueString()
			credentialsConfig["username"] = credentials.Username.ValueString()
			credentialsConfig["password"] = credentials.Password.ValueString()
			credentialsConfig["sslEnabled"] = credentials.SslEnabled.ValueBool()
			credentialsConfig["sslRejectUnauthorized"] = credentials.SslRejectUnauthorized.ValueBool()
			credentialsConfig["sslCertificate"] = credentials.SslCertificate.ValueString()

			return credentialsConfig, diags
		},
		ReadCredentialsForUpdateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel, state AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			credentialsConfig := make(map[string]any)

			var credentialsFromPlan AppConnectionOracleCredentialsModel
			diags := plan.Credentials.As(ctx, &credentialsFromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var credentialsFromState AppConnectionOracleCredentialsModel
			diags = state.Credentials.As(ctx, &credentialsFromState, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != AppConnectionOracleAuthMethodUsernameAndPassword {
				diags.AddError(
					"Unable to update Oracle app connection",
					"Invalid method. Only username-and-password method is supported",
				)
				return nil, diags
			}

			if credentialsFromState.Host.ValueString() != credentialsFromPlan.Host.ValueString() {
				credentialsConfig["host"] = credentialsFromPlan.Host.ValueString()
			}
			if credentialsFromState.Port.ValueInt32() != credentialsFromPlan.Port.ValueInt32() {
				credentialsConfig["port"] = credentialsFromPlan.Port.ValueInt32()
			}
			if credentialsFromState.ServiceName.ValueString() != credentialsFromPlan.ServiceName.ValueString() {
				credentialsConfig["serviceName"] = credentialsFromPlan.ServiceName.ValueString()
			}
			if credentialsFromState.Username.ValueString() != credentialsFromPlan.Username.ValueString() {
				credentialsConfig["username"] = credentialsFromPlan.Username.ValueString()
			}
			if credentialsFromState.Password.ValueString() != credentialsFromPlan.Password.ValueString() {
				credentialsConfig["password"] = credentialsFromPlan.Password.ValueString()
			}
			if credentialsFromState.SslEnabled.ValueBool() != credentialsFromPlan.SslEnabled.ValueBool() {
				credentialsConfig["sslEnabled"] = credentialsFromPlan.SslEnabled.ValueBool()
			}
			if credentialsFromState.SslRejectUnauthorized.ValueBool() != credentialsFromPlan.SslRejectUnauthorized.ValueBool() {
				credentialsConfig["sslRejectUnauthorized"] = credentialsFromPlan.SslRejectUnauthorized.ValueBool()
			}
			if credentialsFromState.SslCertificate.ValueString() != credentialsFromPlan.SslCertificate.ValueString() {
				credentialsConfig["sslCertificate"] = credentialsFromPlan.SslCertificate.ValueString()
			}

			return credentialsConfig, diags
		},
		OverwriteCredentialsFields: func(state *AppConnectionBaseResourceModel) diag.Diagnostics {
			credentialsConfig := map[string]attr.Value{
				"host":                    types.StringNull(),
				"port":                    types.Int32Null(),
				"service_name":            types.StringNull(),
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
				"service_name":            types.StringType,
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