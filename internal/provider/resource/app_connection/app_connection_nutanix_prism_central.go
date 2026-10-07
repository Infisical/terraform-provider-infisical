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

type AppConnectionNutanixPrismCentralCredentialsModel struct {
	Hostname              types.String `tfsdk:"hostname"`
	Port                  types.Int32  `tfsdk:"port"`
	SslRejectUnauthorized types.Bool   `tfsdk:"ssl_reject_unauthorized"`
	SslCertificate        types.String `tfsdk:"ssl_certificate"`
	ApiKey                types.String `tfsdk:"api_key"`
	Username              types.String `tfsdk:"username"`
	Password              types.String `tfsdk:"password"`
}

const NutanixPrismCentralAppConnectionApiKeyMethod = "api-key"
const NutanixPrismCentralAppConnectionBasicAuthMethod = "basic-auth"
const nutanixPrismCentralHostnameForbiddenChars = "/@?:#"
const nutanixPrismCentralApiKeyLength = 32

func isNutanixPrismCentralApiKey(value string) bool {
	if len(value) != nutanixPrismCentralApiKeyLength {
		return false
	}
	for _, c := range value {
		if !(c >= '0' && c <= '9') && !(c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func buildNutanixPrismCentralCredentials(credentials AppConnectionNutanixPrismCentralCredentialsModel, method string, errorSummary string) (map[string]any, diag.Diagnostics) {
	var diags diag.Diagnostics
	credentialsConfig := make(map[string]any)

	if credentials.Hostname.IsNull() || credentials.Hostname.ValueString() == "" {
		diags.AddError(errorSummary, "Hostname field must be defined for all methods")
		return nil, diags
	}

	if strings.ContainsAny(credentials.Hostname.ValueString(), nutanixPrismCentralHostnameForbiddenChars) {
		diags.AddError(errorSummary, "Hostname must not contain any of the following characters: "+nutanixPrismCentralHostnameForbiddenChars)
		return nil, diags
	}

	credentialsConfig["hostname"] = credentials.Hostname.ValueString()

	if !credentials.Port.IsNull() && !credentials.Port.IsUnknown() {
		credentialsConfig["port"] = credentials.Port.ValueInt32()
	}

	if !credentials.SslRejectUnauthorized.IsNull() && !credentials.SslRejectUnauthorized.IsUnknown() {
		credentialsConfig["sslRejectUnauthorized"] = credentials.SslRejectUnauthorized.ValueBool()
	}

	if !credentials.SslCertificate.IsNull() && credentials.SslCertificate.ValueString() != "" {
		credentialsConfig["sslCertificate"] = credentials.SslCertificate.ValueString()
	}

	if method == NutanixPrismCentralAppConnectionApiKeyMethod {
		if credentials.ApiKey.IsNull() || credentials.ApiKey.ValueString() == "" {
			diags.AddError(errorSummary, "API key field must be defined in api-key method")
			return nil, diags
		}

		if !isNutanixPrismCentralApiKey(credentials.ApiKey.ValueString()) {
			diags.AddError(errorSummary, "API key must be a 32-character lowercase hexadecimal string")
			return nil, diags
		}

		credentialsConfig["apiKey"] = credentials.ApiKey.ValueString()
	} else {
		if credentials.Username.IsNull() || credentials.Username.ValueString() == "" {
			diags.AddError(errorSummary, "Username field must be defined in basic-auth method")
			return nil, diags
		}

		if credentials.Password.IsNull() || credentials.Password.ValueString() == "" {
			diags.AddError(errorSummary, "Password field must be defined in basic-auth method")
			return nil, diags
		}

		credentialsConfig["username"] = credentials.Username.ValueString()
		credentialsConfig["password"] = credentials.Password.ValueString()
	}

	return credentialsConfig, diags
}

func NewAppConnectionNutanixPrismCentralResource() resource.Resource {
	return &AppConnectionBaseResource{
		App:               infisical.AppConnectionAppNutanixPrismCentral,
		AppConnectionName: "Nutanix Prism Central",
		ResourceTypeName:  "_app_connection_nutanix_prism_central",
		SupportsGateway:   true,
		AllowedMethods:    []string{NutanixPrismCentralAppConnectionApiKeyMethod, NutanixPrismCentralAppConnectionBasicAuthMethod},
		CredentialsAttributes: map[string]schema.Attribute{
			"hostname": schema.StringAttribute{
				Required:    true,
				Description: "The hostname or IP address of Nutanix Prism Central, without scheme, port or path.",
			},
			"port": schema.Int32Attribute{
				Optional:    true,
				Description: "The port of Nutanix Prism Central.",
			},
			"ssl_reject_unauthorized": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether or not to reject untrusted TLS certificates presented by Nutanix Prism Central.",
			},
			"ssl_certificate": schema.StringAttribute{
				Optional:    true,
				Description: "The PEM-encoded CA certificate used to verify the Nutanix Prism Central TLS certificate.",
			},
			"api_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "The 32-character lowercase hexadecimal Nutanix Prism Central API key, required for the `api-key` method.",
			},
			"username": schema.StringAttribute{
				Optional:    true,
				Description: "The username to authenticate to Nutanix Prism Central with, required for the `basic-auth` method.",
			},
			"password": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "The password to authenticate to Nutanix Prism Central with, required for the `basic-auth` method.",
			},
		},
		ReadCredentialsForCreateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			var credentials AppConnectionNutanixPrismCentralCredentialsModel
			diags := plan.Credentials.As(ctx, &credentials, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			return buildNutanixPrismCentralCredentials(credentials, plan.Method.ValueString(), "Unable to create Nutanix Prism Central app connection")
		},
		ReadCredentialsForUpdateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel, state AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			var credentialsFromPlan AppConnectionNutanixPrismCentralCredentialsModel
			diags := plan.Credentials.As(ctx, &credentialsFromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var credentialsFromState AppConnectionNutanixPrismCentralCredentialsModel
			diags = state.Credentials.As(ctx, &credentialsFromState, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			credentials := credentialsFromPlan
			if credentialsFromPlan.Hostname.IsUnknown() {
				credentials.Hostname = credentialsFromState.Hostname
			}
			if credentialsFromPlan.Port.IsUnknown() {
				credentials.Port = credentialsFromState.Port
			}
			if credentialsFromPlan.SslRejectUnauthorized.IsUnknown() {
				credentials.SslRejectUnauthorized = credentialsFromState.SslRejectUnauthorized
			}
			if credentialsFromPlan.SslCertificate.IsUnknown() {
				credentials.SslCertificate = credentialsFromState.SslCertificate
			}
			if credentialsFromPlan.ApiKey.IsUnknown() {
				credentials.ApiKey = credentialsFromState.ApiKey
			}
			if credentialsFromPlan.Username.IsUnknown() {
				credentials.Username = credentialsFromState.Username
			}
			if credentialsFromPlan.Password.IsUnknown() {
				credentials.Password = credentialsFromState.Password
			}

			return buildNutanixPrismCentralCredentials(credentials, plan.Method.ValueString(), "Unable to update Nutanix Prism Central app connection")
		},
		OverwriteCredentialsFields: func(state *AppConnectionBaseResourceModel) diag.Diagnostics {
			credentialsConfig := map[string]attr.Value{
				"hostname":                types.StringNull(),
				"port":                    types.Int32Null(),
				"ssl_reject_unauthorized": types.BoolNull(),
				"ssl_certificate":         types.StringNull(),
				"api_key":                 types.StringNull(),
				"username":                types.StringNull(),
				"password":                types.StringNull(),
			}

			var diags diag.Diagnostics
			state.Credentials, diags = types.ObjectValue(map[string]attr.Type{
				"hostname":                types.StringType,
				"port":                    types.Int32Type,
				"ssl_reject_unauthorized": types.BoolType,
				"ssl_certificate":         types.StringType,
				"api_key":                 types.StringType,
				"username":                types.StringType,
				"password":                types.StringType,
			}, credentialsConfig)

			return diags
		},
	}
}
