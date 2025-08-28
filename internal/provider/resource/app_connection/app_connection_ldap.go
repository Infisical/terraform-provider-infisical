package resource

import (
	"context"
	"strings"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// AppConnectionLdapCredentialsModel describes the data source data model.
type AppConnectionLdapCredentialsModel struct {
	Provider              types.String `tfsdk:"provider"`
	Url                   types.String `tfsdk:"url"`
	Dn                    types.String `tfsdk:"dn"`
	Password              types.String `tfsdk:"password"`
	SslRejectUnauthorized types.Bool   `tfsdk:"ssl_reject_unauthorized"`
	SslCertificate        types.String `tfsdk:"ssl_certificate"`
}

const AppConnectionLdapAuthMethodSimpleBind = "simple-bind"

func NewAppConnectionLdapResource() resource.Resource {
	return &AppConnectionBaseResource{
		App:               infisical.AppConnectionAppLdap,
		AppConnectionName: "LDAP",
		ResourceTypeName:  "_app_connection_ldap",
		AllowedMethods:    []string{AppConnectionLdapAuthMethodSimpleBind},
		CredentialsAttributes: map[string]schema.Attribute{
			"provider": schema.StringAttribute{
				Required:    true,
				Description: "The LDAP provider (e.g., 'active-directory').",
			},
			"url": schema.StringAttribute{
				Required:    true,
				Description: "The LDAP server URL (e.g., 'ldap://example.com:389' or 'ldaps://example.com:636').",
			},
			"dn": schema.StringAttribute{
				Required:    true,
				Description: "The Distinguished Name (DN) or User Principal Name (UPN) of the principal to bind with (e.g., 'CN=John,CN=Users,DC=example,DC=com').",
			},
			"password": schema.StringAttribute{
				Required:    true,
				Description: "The password to bind with for authentication.",
				Sensitive:   true,
			},
			"ssl_reject_unauthorized": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether or not to reject unauthorized SSL certificates (true/false) when using ldaps://. Set to false only in test environments.",
				Default:     booldefault.StaticBool(true),
			},
			"ssl_certificate": schema.StringAttribute{
				Optional:    true,
				Description: "The SSL certificate (PEM format) to use for secure connection when using ldaps:// with a self-signed certificate.",
			},
		},
		ReadCredentialsForCreateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			credentialsConfig := make(map[string]any)

			var credentials AppConnectionLdapCredentialsModel
			diags := plan.Credentials.As(ctx, &credentials, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != AppConnectionLdapAuthMethodSimpleBind {
				diags.AddError(
					"Unable to create LDAP app connection",
					"Invalid method. Only simple-bind method is supported",
				)
				return nil, diags
			}

			credentialsConfig["provider"] = credentials.Provider.ValueString()
			credentialsConfig["url"] = credentials.Url.ValueString()
			credentialsConfig["dn"] = credentials.Dn.ValueString()
			credentialsConfig["password"] = credentials.Password.ValueString()

			// Validate SSL settings based on URL scheme
			url := credentials.Url.ValueString()
			isLdaps := strings.HasPrefix(strings.ToLower(url), "ldaps://")

			if !isLdaps {
				// For ldap:// URLs, SSL is not used
				if !credentials.SslRejectUnauthorized.IsNull() && credentials.SslRejectUnauthorized.ValueBool() {
					diags.AddError(
						"Invalid SSL configuration",
						"ssl_reject_unauthorized cannot be true for ldap:// URLs since they don't use SSL. Use ldaps:// for secure connections or set ssl_reject_unauthorized to false.",
					)
					return nil, diags
				}
				// Default to false for ldap:// URLs
				credentialsConfig["sslRejectUnauthorized"] = false
			} else {
				credentialsConfig["sslRejectUnauthorized"] = credentials.SslRejectUnauthorized.ValueBool()
			}

			if !credentials.SslCertificate.IsNull() {
				credentialsConfig["sslCertificate"] = credentials.SslCertificate.ValueString()
			}

			return credentialsConfig, diags
		},
		ReadCredentialsForUpdateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel, state AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			credentialsConfig := make(map[string]any)

			var credentialsFromPlan AppConnectionLdapCredentialsModel
			diags := plan.Credentials.As(ctx, &credentialsFromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var credentialsFromState AppConnectionLdapCredentialsModel
			diags = state.Credentials.As(ctx, &credentialsFromState, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != AppConnectionLdapAuthMethodSimpleBind {
				diags.AddError(
					"Unable to update LDAP app connection",
					"Invalid method. Only simple-bind method is supported",
				)
				return nil, diags
			}

			credentialsConfig["provider"] = credentialsFromPlan.Provider.ValueString()
			credentialsConfig["url"] = credentialsFromPlan.Url.ValueString()
			credentialsConfig["dn"] = credentialsFromPlan.Dn.ValueString()
			credentialsConfig["password"] = credentialsFromPlan.Password.ValueString()

			// Validate SSL settings based on URL scheme for updates
			url := credentialsFromPlan.Url.ValueString()
			isLdaps := strings.HasPrefix(strings.ToLower(url), "ldaps://")

			if !isLdaps {
				// For ldap:// URLs, SSL is not used
				if !credentialsFromPlan.SslRejectUnauthorized.IsNull() && credentialsFromPlan.SslRejectUnauthorized.ValueBool() {
					diags.AddError(
						"Invalid SSL configuration",
						"ssl_reject_unauthorized cannot be true for ldap:// URLs since they don't use SSL. Use ldaps:// for secure connections or set ssl_reject_unauthorized to false.",
					)
					return nil, diags
				}
				// Always set to false for ldap:// URLs, regardless of state
				credentialsConfig["sslRejectUnauthorized"] = false
			} else {
				credentialsConfig["sslRejectUnauthorized"] = credentialsFromPlan.SslRejectUnauthorized.ValueBool()
			}

			if !credentialsFromPlan.SslCertificate.IsNull() {
				credentialsConfig["sslCertificate"] = credentialsFromPlan.SslCertificate.ValueString()
			}

			return credentialsConfig, diags
		},
		OverwriteCredentialsFields: func(state *AppConnectionBaseResourceModel) diag.Diagnostics {
			credentialsConfig := map[string]attr.Value{
				"provider":                types.StringNull(),
				"url":                     types.StringNull(),
				"dn":                      types.StringNull(),
				"password":                types.StringNull(),
				"ssl_reject_unauthorized": types.BoolNull(),
				"ssl_certificate":         types.StringNull(),
			}

			var diags diag.Diagnostics
			state.Credentials, diags = types.ObjectValue(map[string]attr.Type{
				"provider":                types.StringType,
				"url":                     types.StringType,
				"dn":                      types.StringType,
				"password":                types.StringType,
				"ssl_reject_unauthorized": types.BoolType,
				"ssl_certificate":         types.StringType,
			}, credentialsConfig)

			return diags
		},
	}
}
