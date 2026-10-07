package resource

import (
	"context"
	"strings"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type AppConnectionNetScalerCredentialsModel struct {
	Hostname              types.String `tfsdk:"hostname"`
	Port                  types.Int32  `tfsdk:"port"`
	Username              types.String `tfsdk:"username"`
	Password              types.String `tfsdk:"password"`
	SslRejectUnauthorized types.Bool   `tfsdk:"ssl_reject_unauthorized"`
	SslCertificate        types.String `tfsdk:"ssl_certificate"`
}

const NetScalerAppConnectionBasicAuthMethod = "basic-auth"
const netScalerHostnameForbiddenChars = "/@?"

func NewAppConnectionNetScalerResource() resource.Resource {
	return &AppConnectionBaseResource{
		App:               infisical.AppConnectionAppNetScaler,
		AppConnectionName: "NetScaler",
		ResourceTypeName:  "_app_connection_netscaler",
		SupportsGateway:   true,
		AllowedMethods:    []string{NetScalerAppConnectionBasicAuthMethod},
		CredentialsAttributes: map[string]schema.Attribute{
			"hostname": schema.StringAttribute{
				Required:    true,
				Description: "The hostname or IP address of the NetScaler management interface, without scheme or path.",
			},
			"port": schema.Int32Attribute{
				Optional:    true,
				Description: "The port of the NetScaler management interface.",
			},
			"username": schema.StringAttribute{
				Required:    true,
				Description: "The username to authenticate to NetScaler with.",
			},
			"password": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "The password to authenticate to NetScaler with.",
			},
			"ssl_reject_unauthorized": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether or not to reject untrusted TLS certificates presented by NetScaler.",
			},
			"ssl_certificate": schema.StringAttribute{
				Optional:    true,
				Description: "The PEM-encoded CA certificate used to verify the NetScaler TLS certificate.",
			},
		},
		ReadCredentialsForCreateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			credentialsConfig := make(map[string]any)

			var credentials AppConnectionNetScalerCredentialsModel
			diags := plan.Credentials.As(ctx, &credentials, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != NetScalerAppConnectionBasicAuthMethod {
				diags.AddError(
					"Unable to create NetScaler app connection",
					"Invalid method. Only basic-auth method is supported",
				)
				return nil, diags
			}

			if credentials.Hostname.IsNull() || credentials.Hostname.ValueString() == "" {
				diags.AddError(
					"Unable to create NetScaler app connection",
					"Hostname field must be defined in basic-auth method",
				)
				return nil, diags
			}

			if strings.ContainsAny(credentials.Hostname.ValueString(), netScalerHostnameForbiddenChars) {
				diags.AddError(
					"Unable to create NetScaler app connection",
					"Hostname must not contain any of the following characters: "+netScalerHostnameForbiddenChars,
				)
				return nil, diags
			}

			if credentials.Username.IsNull() || credentials.Username.ValueString() == "" {
				diags.AddError(
					"Unable to create NetScaler app connection",
					"Username field must be defined in basic-auth method",
				)
				return nil, diags
			}

			if credentials.Password.IsNull() || credentials.Password.ValueString() == "" {
				diags.AddError(
					"Unable to create NetScaler app connection",
					"Password field must be defined in basic-auth method",
				)
				return nil, diags
			}

			credentialsConfig["hostname"] = credentials.Hostname.ValueString()
			credentialsConfig["username"] = credentials.Username.ValueString()
			credentialsConfig["password"] = credentials.Password.ValueString()

			if !credentials.Port.IsNull() && !credentials.Port.IsUnknown() {
				credentialsConfig["port"] = credentials.Port.ValueInt32()
			}

			if !credentials.SslRejectUnauthorized.IsNull() && !credentials.SslRejectUnauthorized.IsUnknown() {
				credentialsConfig["sslRejectUnauthorized"] = credentials.SslRejectUnauthorized.ValueBool()
			}

			if !credentials.SslCertificate.IsNull() && credentials.SslCertificate.ValueString() != "" {
				credentialsConfig["sslCertificate"] = credentials.SslCertificate.ValueString()
			}

			return credentialsConfig, diags
		},
		ReadCredentialsForUpdateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel, state AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			credentialsConfig := make(map[string]any)

			var credentialsFromPlan AppConnectionNetScalerCredentialsModel
			diags := plan.Credentials.As(ctx, &credentialsFromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var credentialsFromState AppConnectionNetScalerCredentialsModel
			diags = state.Credentials.As(ctx, &credentialsFromState, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != NetScalerAppConnectionBasicAuthMethod {
				diags.AddError(
					"Unable to update NetScaler app connection",
					"Invalid method. Only basic-auth method is supported",
				)
				return nil, diags
			}

			hostname := credentialsFromPlan.Hostname
			if credentialsFromPlan.Hostname.IsUnknown() {
				hostname = credentialsFromState.Hostname
			}

			port := credentialsFromPlan.Port
			if credentialsFromPlan.Port.IsUnknown() {
				port = credentialsFromState.Port
			}

			username := credentialsFromPlan.Username
			if credentialsFromPlan.Username.IsUnknown() {
				username = credentialsFromState.Username
			}

			password := credentialsFromPlan.Password
			if credentialsFromPlan.Password.IsUnknown() {
				password = credentialsFromState.Password
			}

			sslRejectUnauthorized := credentialsFromPlan.SslRejectUnauthorized
			if credentialsFromPlan.SslRejectUnauthorized.IsUnknown() {
				sslRejectUnauthorized = credentialsFromState.SslRejectUnauthorized
			}

			sslCertificate := credentialsFromPlan.SslCertificate
			if credentialsFromPlan.SslCertificate.IsUnknown() {
				sslCertificate = credentialsFromState.SslCertificate
			}

			if hostname.IsNull() || hostname.ValueString() == "" {
				diags.AddError(
					"Unable to update NetScaler app connection",
					"Hostname field must be defined in basic-auth method",
				)
				return nil, diags
			}

			if strings.ContainsAny(hostname.ValueString(), netScalerHostnameForbiddenChars) {
				diags.AddError(
					"Unable to update NetScaler app connection",
					"Hostname must not contain any of the following characters: "+netScalerHostnameForbiddenChars,
				)
				return nil, diags
			}

			if username.IsNull() || username.ValueString() == "" {
				diags.AddError(
					"Unable to update NetScaler app connection",
					"Username field must be defined in basic-auth method",
				)
				return nil, diags
			}

			if password.IsNull() || password.ValueString() == "" {
				diags.AddError(
					"Unable to update NetScaler app connection",
					"Password field must be defined in basic-auth method",
				)
				return nil, diags
			}

			credentialsConfig["hostname"] = hostname.ValueString()
			credentialsConfig["username"] = username.ValueString()
			credentialsConfig["password"] = password.ValueString()

			if !port.IsNull() && !port.IsUnknown() {
				credentialsConfig["port"] = port.ValueInt32()
			}

			if !sslRejectUnauthorized.IsNull() && !sslRejectUnauthorized.IsUnknown() {
				credentialsConfig["sslRejectUnauthorized"] = sslRejectUnauthorized.ValueBool()
			}

			if !sslCertificate.IsNull() && sslCertificate.ValueString() != "" {
				credentialsConfig["sslCertificate"] = sslCertificate.ValueString()
			}

			return credentialsConfig, diags
		},
		OverwriteCredentialsFields: func(state *AppConnectionBaseResourceModel) diag.Diagnostics {
			credentialsConfig := map[string]attr.Value{
				"hostname":                types.StringNull(),
				"port":                    types.Int32Null(),
				"username":                types.StringNull(),
				"password":                types.StringNull(),
				"ssl_reject_unauthorized": types.BoolNull(),
				"ssl_certificate":         types.StringNull(),
			}

			var diags diag.Diagnostics
			state.Credentials, diags = types.ObjectValue(map[string]attr.Type{
				"hostname":                types.StringType,
				"port":                    types.Int32Type,
				"username":                types.StringType,
				"password":                types.StringType,
				"ssl_reject_unauthorized": types.BoolType,
				"ssl_certificate":         types.StringType,
			}, credentialsConfig)

			return diags
		},
	}
}
