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

type AppConnectionWinRMCredentialsModel struct {
	Host                  types.String `tfsdk:"host"`
	Port                  types.Int32  `tfsdk:"port"`
	Username              types.String `tfsdk:"username"`
	Password              types.String `tfsdk:"password"`
	SslEnabled            types.Bool   `tfsdk:"ssl_enabled"`
	SslRejectUnauthorized types.Bool   `tfsdk:"ssl_reject_unauthorized"`
	SslCertificate        types.String `tfsdk:"ssl_certificate"`
}

const WinRMAppConnectionUsernamePasswordMethod = "username-password"
const winRMAppConnectionDefaultPort = 5985
const winRMAppConnectionDefaultSslEnabled = false
const winRMAppConnectionDefaultSslRejectUnauthorized = true

func buildWinRMCredentials(credentials AppConnectionWinRMCredentialsModel, method string, errorSummary string) (map[string]any, diag.Diagnostics) {
	var diags diag.Diagnostics
	credentialsConfig := make(map[string]any)

	if method != WinRMAppConnectionUsernamePasswordMethod {
		diags.AddError(errorSummary, "Invalid method. Only username-password method is supported")
		return nil, diags
	}

	if credentials.Host.IsNull() || credentials.Host.ValueString() == "" {
		diags.AddError(errorSummary, "Host field must be defined in username-password method")
		return nil, diags
	}

	if credentials.Username.IsNull() || credentials.Username.ValueString() == "" {
		diags.AddError(errorSummary, "Username field must be defined in username-password method")
		return nil, diags
	}

	if credentials.Password.IsNull() || credentials.Password.ValueString() == "" {
		diags.AddError(errorSummary, "Password field must be defined in username-password method")
		return nil, diags
	}

	port := int32(winRMAppConnectionDefaultPort)
	if !credentials.Port.IsNull() && !credentials.Port.IsUnknown() {
		port = credentials.Port.ValueInt32()
	}

	sslEnabled := winRMAppConnectionDefaultSslEnabled
	if !credentials.SslEnabled.IsNull() && !credentials.SslEnabled.IsUnknown() {
		sslEnabled = credentials.SslEnabled.ValueBool()
	}

	sslRejectUnauthorized := winRMAppConnectionDefaultSslRejectUnauthorized
	if !credentials.SslRejectUnauthorized.IsNull() && !credentials.SslRejectUnauthorized.IsUnknown() {
		sslRejectUnauthorized = credentials.SslRejectUnauthorized.ValueBool()
	}

	credentialsConfig["host"] = credentials.Host.ValueString()
	credentialsConfig["port"] = port
	credentialsConfig["username"] = credentials.Username.ValueString()
	credentialsConfig["password"] = credentials.Password.ValueString()
	credentialsConfig["sslEnabled"] = sslEnabled
	credentialsConfig["sslRejectUnauthorized"] = sslRejectUnauthorized

	if !credentials.SslCertificate.IsNull() && credentials.SslCertificate.ValueString() != "" {
		credentialsConfig["sslCertificate"] = credentials.SslCertificate.ValueString()
	}

	return credentialsConfig, diags
}

func NewAppConnectionWinRMResource() resource.Resource {
	return &AppConnectionBaseResource{
		App:               infisical.AppConnectionAppWinRM,
		AppConnectionName: "Windows (WinRM)",
		ResourceTypeName:  "_app_connection_winrm",
		SupportsGateway:   true,
		RequiresGateway:   true,
		AllowedMethods:    []string{WinRMAppConnectionUsernamePasswordMethod},
		CredentialsAttributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Required:    true,
				Description: "The DNS name (FQDN) or IP address of the Windows host.",
			},
			"port": schema.Int32Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int32default.StaticInt32(winRMAppConnectionDefaultPort),
				Description: "The WinRM port, usually 5985 for HTTP and 5986 for HTTPS.",
			},
			"username": schema.StringAttribute{
				Required:    true,
				Description: "The Windows account to authenticate as, in the form `DOMAIN\\user` or `user@domain`.",
			},
			"password": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "The password of the Windows account.",
			},
			"ssl_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(winRMAppConnectionDefaultSslEnabled),
				Description: "Whether or not to connect over HTTPS instead of HTTP with NTLM message encryption.",
			},
			"ssl_reject_unauthorized": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(winRMAppConnectionDefaultSslRejectUnauthorized),
				Description: "Whether or not to reject untrusted TLS certificates presented by the WinRM HTTPS listener.",
			},
			"ssl_certificate": schema.StringAttribute{
				Optional:    true,
				Description: "The PEM-encoded CA certificate used to verify a self-signed WinRM HTTPS listener.",
			},
		},
		ReadCredentialsForCreateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			var credentials AppConnectionWinRMCredentialsModel
			diags := plan.Credentials.As(ctx, &credentials, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			return buildWinRMCredentials(credentials, plan.Method.ValueString(), "Unable to create Windows (WinRM) app connection")
		},
		ReadCredentialsForUpdateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel, state AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			var credentialsFromPlan AppConnectionWinRMCredentialsModel
			diags := plan.Credentials.As(ctx, &credentialsFromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var credentialsFromState AppConnectionWinRMCredentialsModel
			diags = state.Credentials.As(ctx, &credentialsFromState, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			credentials := credentialsFromPlan
			if credentialsFromPlan.Host.IsUnknown() {
				credentials.Host = credentialsFromState.Host
			}
			if credentialsFromPlan.Port.IsUnknown() {
				credentials.Port = credentialsFromState.Port
			}
			if credentialsFromPlan.Username.IsUnknown() {
				credentials.Username = credentialsFromState.Username
			}
			if credentialsFromPlan.Password.IsUnknown() {
				credentials.Password = credentialsFromState.Password
			}
			if credentialsFromPlan.SslEnabled.IsUnknown() {
				credentials.SslEnabled = credentialsFromState.SslEnabled
			}
			if credentialsFromPlan.SslRejectUnauthorized.IsUnknown() {
				credentials.SslRejectUnauthorized = credentialsFromState.SslRejectUnauthorized
			}
			if credentialsFromPlan.SslCertificate.IsUnknown() {
				credentials.SslCertificate = credentialsFromState.SslCertificate
			}

			return buildWinRMCredentials(credentials, plan.Method.ValueString(), "Unable to update Windows (WinRM) app connection")
		},
		OverwriteCredentialsFields: func(state *AppConnectionBaseResourceModel) diag.Diagnostics {
			credentialsConfig := map[string]attr.Value{
				"host":                    types.StringNull(),
				"port":                    types.Int32Null(),
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
