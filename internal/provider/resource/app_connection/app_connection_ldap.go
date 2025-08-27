package resource

import (
	"context"
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
				Description: "The LDAP provider (e.g., 'active-directory', 'openldap').",
			},
			"url": schema.StringAttribute{
				Required:    true,
				Description: "The LDAP server URL (e.g., 'ldap://example.com:389' or 'ldaps://example.com:636').",
			},
			"dn": schema.StringAttribute{
				Required:    true,
				Description: "The Distinguished Name (DN) for authentication.",
			},
			"password": schema.StringAttribute{
				Required:    true,
				Description: "The password for authentication.",
				Sensitive:   true,
			},
			"ssl_reject_unauthorized": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to reject unauthorized SSL certificates.",
				Default:     booldefault.StaticBool(true),
			},
			"ssl_certificate": schema.StringAttribute{
				Optional:    true,
				Description: "The SSL certificate for secure connections.",
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
			credentialsConfig["sslRejectUnauthorized"] = credentials.SslRejectUnauthorized.ValueBool()

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

			if credentialsFromState.Provider.ValueString() != credentialsFromPlan.Provider.ValueString() {
				credentialsConfig["provider"] = credentialsFromPlan.Provider.ValueString()
			}
			if credentialsFromState.Url.ValueString() != credentialsFromPlan.Url.ValueString() {
				credentialsConfig["url"] = credentialsFromPlan.Url.ValueString()
			}
			if credentialsFromState.Dn.ValueString() != credentialsFromPlan.Dn.ValueString() {
				credentialsConfig["dn"] = credentialsFromPlan.Dn.ValueString()
			}
			if credentialsFromState.Password.ValueString() != credentialsFromPlan.Password.ValueString() {
				credentialsConfig["password"] = credentialsFromPlan.Password.ValueString()
			}
			if credentialsFromState.SslRejectUnauthorized.ValueBool() != credentialsFromPlan.SslRejectUnauthorized.ValueBool() {
				credentialsConfig["sslRejectUnauthorized"] = credentialsFromPlan.SslRejectUnauthorized.ValueBool()
			}
			if credentialsFromState.SslCertificate.ValueString() != credentialsFromPlan.SslCertificate.ValueString() {
				if !credentialsFromPlan.SslCertificate.IsNull() {
					credentialsConfig["sslCertificate"] = credentialsFromPlan.SslCertificate.ValueString()
				} else {
					credentialsConfig["sslCertificate"] = ""
				}
			}

			return credentialsConfig, diags
		},
		OverwriteCredentialsFields: func(state *AppConnectionBaseResourceModel) diag.Diagnostics {
			var diags diag.Diagnostics

			// Parse existing credentials
			var credentials AppConnectionLdapCredentialsModel
			diags = state.Credentials.As(context.Background(), &credentials, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return diags
			}

			// Create the credentials object with the same structure
			credentialsObject, objDiags := types.ObjectValue(
				map[string]attr.Type{
					"provider":                types.StringType,
					"url":                     types.StringType,
					"dn":                      types.StringType,
					"password":                types.StringType,
					"ssl_reject_unauthorized": types.BoolType,
					"ssl_certificate":         types.StringType,
				},
				map[string]attr.Value{
					"provider":                credentials.Provider,
					"url":                     credentials.Url,
					"dn":                      credentials.Dn,
					"password":                credentials.Password,
					"ssl_reject_unauthorized": credentials.SslRejectUnauthorized,
					"ssl_certificate":         credentials.SslCertificate,
				},
			)
			diags.Append(objDiags...)
			if diags.HasError() {
				return diags
			}

			state.Credentials = credentialsObject
			return diags
		},
	}
}
