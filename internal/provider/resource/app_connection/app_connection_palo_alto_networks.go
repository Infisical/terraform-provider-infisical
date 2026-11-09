package resource

import (
	"context"
	"regexp"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type AppConnectionPaloAltoNetworksCredentialsModel struct {
	Hostname              types.String `tfsdk:"hostname"`
	Port                  types.Int32  `tfsdk:"port"`
	Username              types.String `tfsdk:"username"`
	Password              types.String `tfsdk:"password"`
	SslRejectUnauthorized types.Bool   `tfsdk:"ssl_reject_unauthorized"`
	SslCertificate        types.String `tfsdk:"ssl_certificate"`
}

const PaloAltoNetworksAppConnectionBasicAuthMethod = "basic-auth"

var paloAltoNetworksHostnamePattern = regexp.MustCompile(`^[^/@?]*$`)

func paloAltoNetworksCredentialsPayload(credentials AppConnectionPaloAltoNetworksCredentialsModel) map[string]any {
	credentialsConfig := map[string]any{
		"hostname": credentials.Hostname.ValueString(),
		"username": credentials.Username.ValueString(),
		"password": credentials.Password.ValueString(),
	}

	if !credentials.Port.IsNull() && !credentials.Port.IsUnknown() {
		credentialsConfig["port"] = credentials.Port.ValueInt32()
	}

	if !credentials.SslRejectUnauthorized.IsNull() && !credentials.SslRejectUnauthorized.IsUnknown() {
		credentialsConfig["sslRejectUnauthorized"] = credentials.SslRejectUnauthorized.ValueBool()
	}

	if !credentials.SslCertificate.IsNull() && credentials.SslCertificate.ValueString() != "" {
		credentialsConfig["sslCertificate"] = credentials.SslCertificate.ValueString()
	}

	return credentialsConfig
}

func NewAppConnectionPaloAltoNetworksResource() resource.Resource {
	return &AppConnectionBaseResource{
		App:               infisical.AppConnectionAppPaloAltoNetworks,
		AppConnectionName: "Palo Alto Networks",
		ResourceTypeName:  "_app_connection_palo_alto_networks",
		SupportsGateway:   true,
		AllowedMethods:    []string{PaloAltoNetworksAppConnectionBasicAuthMethod},
		CredentialsAttributes: map[string]schema.Attribute{
			"hostname": schema.StringAttribute{
				Required:    true,
				Description: "The hostname or IP address of the PAN-OS firewall or Panorama management interface, without scheme, port or path.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 512),
					stringvalidator.RegexMatches(paloAltoNetworksHostnamePattern, "must not contain /, @, or ? characters"),
				},
			},
			"port": schema.Int32Attribute{
				Optional:    true,
				Description: "The port of the PAN-OS management interface. Defaults to `443`.",
				Validators:  []validator.Int32{int32validator.Between(1, 65535)},
			},
			"username": schema.StringAttribute{
				Required:    true,
				Description: "The username to authenticate to PAN-OS with.",
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 256)},
			},
			"password": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "The password to authenticate to PAN-OS with.",
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 512)},
			},
			"ssl_reject_unauthorized": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether or not to reject untrusted TLS certificates presented by PAN-OS. Defaults to `true`.",
			},
			"ssl_certificate": schema.StringAttribute{
				Optional:    true,
				Description: "The PEM-encoded CA certificate used to verify the PAN-OS TLS certificate.",
				Validators:  []validator.String{stringvalidator.LengthAtMost(16384)},
			},
		},
		ReadCredentialsForCreateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			var credentials AppConnectionPaloAltoNetworksCredentialsModel
			diags := plan.Credentials.As(ctx, &credentials, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			return paloAltoNetworksCredentialsPayload(credentials), diags
		},
		ReadCredentialsForUpdateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel, state AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			var credentialsFromPlan AppConnectionPaloAltoNetworksCredentialsModel
			diags := plan.Credentials.As(ctx, &credentialsFromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var credentialsFromState AppConnectionPaloAltoNetworksCredentialsModel
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
			if credentialsFromPlan.Username.IsUnknown() {
				credentials.Username = credentialsFromState.Username
			}
			if credentialsFromPlan.Password.IsUnknown() {
				credentials.Password = credentialsFromState.Password
			}
			if credentialsFromPlan.SslRejectUnauthorized.IsUnknown() {
				credentials.SslRejectUnauthorized = credentialsFromState.SslRejectUnauthorized
			}
			if credentialsFromPlan.SslCertificate.IsUnknown() {
				credentials.SslCertificate = credentialsFromState.SslCertificate
			}

			return paloAltoNetworksCredentialsPayload(credentials), diags
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
