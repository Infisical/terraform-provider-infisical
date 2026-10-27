package resource

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type PamAccountWindowsAdConnectionDetailsModel struct {
	Domain                  types.String `tfsdk:"domain"`
	DcAddress               types.String `tfsdk:"dc_address"`
	Hosts                   types.List   `tfsdk:"hosts"`
	LdapPort                types.Int64  `tfsdk:"ldap_port"`
	RdpPort                 types.Int64  `tfsdk:"rdp_port"`
	UseLdaps                types.Bool   `tfsdk:"use_ldaps"`
	LdapRejectUnauthorized  types.Bool   `tfsdk:"ldap_reject_unauthorized"`
	LdapCaCert              types.String `tfsdk:"ldap_ca_cert"`
	LdapTlsServerName       types.String `tfsdk:"ldap_tls_server_name"`
	WinrmPort               types.Int64  `tfsdk:"winrm_port"`
	UseWinrmHttps           types.Bool   `tfsdk:"use_winrm_https"`
	WinrmRejectUnauthorized types.Bool   `tfsdk:"winrm_reject_unauthorized"`
	WinrmCaCert             types.String `tfsdk:"winrm_ca_cert"`
}

type PamAccountWindowsAdCredentialsModel struct {
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

func pamPortAttribute(description string, defaultPort int64) schema.Int64Attribute {
	return schema.Int64Attribute{
		Description: description,
		Optional:    true,
		Computed:    true,
		Default:     int64default.StaticInt64(defaultPort),
		Validators:  []validator.Int64{int64validator.Between(1, 65535)},
	}
}

func NewPamAccountWindowsAdResource() resource.Resource {
	return &PamAccountBaseResource{
		AccountType:      "windows-ad",
		AccountTypeName:  "Windows Active Directory",
		ResourceTypeName: "_pam_account_windows_ad",
		SupportsRotation: true,
		ConnectionDetailsAttributes: map[string]schema.Attribute{
			"domain": schema.StringAttribute{
				Description: "The fully qualified domain name of the Active Directory domain, e.g. corp.example.com.",
				Required:    true,
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 255)},
			},
			"dc_address": schema.StringAttribute{
				Description: "The hostname or IP address of a domain controller, used for LDAP connections.",
				Required:    true,
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 255)},
			},
			"hosts": schema.ListAttribute{
				Description: "The hostnames or IP addresses the account may open sessions to.",
				Required:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.ValueStringsAre(
						stringvalidator.LengthBetween(1, 255),
						stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Za-z0-9.:_-]+$`), "must be a valid hostname or IP address"),
					),
				},
			},
			"ldap_port": pamPortAttribute("The LDAP port of the domain controller. Defaults to 389.", 389),
			"rdp_port":  pamPortAttribute("The RDP port of the hosts. Defaults to 3389.", 3389),
			"use_ldaps": schema.BoolAttribute{
				Description: "Whether to connect to the domain controller over LDAPS. Defaults to false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"ldap_reject_unauthorized": schema.BoolAttribute{
				Description: "Whether to reject a domain controller certificate that isn't trusted. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"ldap_ca_cert": schema.StringAttribute{
				Description: "The PEM-encoded CA certificate to verify the domain controller with over LDAPS.",
				Optional:    true,
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"ldap_tls_server_name": schema.StringAttribute{
				Description: "The server name to verify the domain controller's certificate against over LDAPS.",
				Optional:    true,
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"winrm_port": pamPortAttribute("The WinRM port of the hosts, used for credential rotation. Defaults to 5985.", 5985),
			"use_winrm_https": schema.BoolAttribute{
				Description: "Whether to use WinRM over HTTPS. Defaults to false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"winrm_reject_unauthorized": schema.BoolAttribute{
				Description: "Whether to reject a WinRM certificate that isn't trusted. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"winrm_ca_cert": schema.StringAttribute{
				Description: "The PEM-encoded CA certificate to verify WinRM over HTTPS with.",
				Optional:    true,
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
		},
		CredentialsAttributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				Description: "The domain user, in DOMAIN\\username form (e.g. CORP\\Administrator).",
				Required:    true,
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 255)},
			},
			"password": schema.StringAttribute{
				Description: "The password of the domain user.",
				Optional:    true,
				Sensitive:   true,
			},
		},

		ConnectionDetailsToAPI: func(ctx context.Context, connectionDetails types.Object) (map[string]any, diag.Diagnostics) {
			var model PamAccountWindowsAdConnectionDetailsModel
			diags := connectionDetails.As(ctx, &model, basetypes.ObjectAsOptions{})
			var hosts []string
			diags.Append(model.Hosts.ElementsAs(ctx, &hosts, false)...)
			values := map[string]any{
				"domain":                  model.Domain.ValueString(),
				"dcAddress":               model.DcAddress.ValueString(),
				"hosts":                   hosts,
				"port":                    model.LdapPort.ValueInt64(),
				"rdpPort":                 model.RdpPort.ValueInt64(),
				"useLdaps":                model.UseLdaps.ValueBool(),
				"ldapRejectUnauthorized":  model.LdapRejectUnauthorized.ValueBool(),
				"winrmPort":               model.WinrmPort.ValueInt64(),
				"useWinrmHttps":           model.UseWinrmHttps.ValueBool(),
				"winrmRejectUnauthorized": model.WinrmRejectUnauthorized.ValueBool(),
			}
			setString(values, "ldapCaCert", model.LdapCaCert)
			setString(values, "ldapTlsServerName", model.LdapTlsServerName)
			setString(values, "winrmCaCert", model.WinrmCaCert)
			return values, diags
		},

		ConnectionDetailsFromAPI: func(connectionDetails map[string]any) any {
			return PamAccountWindowsAdConnectionDetailsModel{
				Domain:                  apiString(connectionDetails, "domain"),
				DcAddress:               apiString(connectionDetails, "dcAddress"),
				Hosts:                   apiStringList(connectionDetails, "hosts"),
				LdapPort:                apiInt64(connectionDetails, "port"),
				RdpPort:                 apiInt64(connectionDetails, "rdpPort"),
				UseLdaps:                apiBool(connectionDetails, "useLdaps"),
				LdapRejectUnauthorized:  apiBool(connectionDetails, "ldapRejectUnauthorized"),
				LdapCaCert:              apiString(connectionDetails, "ldapCaCert"),
				LdapTlsServerName:       apiString(connectionDetails, "ldapTlsServerName"),
				WinrmPort:               apiInt64(connectionDetails, "winrmPort"),
				UseWinrmHttps:           apiBool(connectionDetails, "useWinrmHttps"),
				WinrmRejectUnauthorized: apiBool(connectionDetails, "winrmRejectUnauthorized"),
				WinrmCaCert:             apiString(connectionDetails, "winrmCaCert"),
			}
		},

		CredentialsToAPI: func(ctx context.Context, credentials types.Object) (map[string]any, diag.Diagnostics) {
			var model PamAccountWindowsAdCredentialsModel
			diags := credentials.As(ctx, &model, basetypes.ObjectAsOptions{})
			values := map[string]any{"username": model.Username.ValueString()}
			setString(values, "password", model.Password)
			return values, diags
		},

		CredentialsFromAPI: func(_ context.Context, credentials map[string]any, prior types.Object) (any, diag.Diagnostics) {
			return PamAccountWindowsAdCredentialsModel{
				Username: apiString(credentials, "username"),
				Password: priorString(prior, "password"),
			}, nil
		},
	}
}
