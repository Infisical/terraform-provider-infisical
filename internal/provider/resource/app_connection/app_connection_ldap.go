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

// AppConnectionLdapCredentialsModel describes the data source data model.
type AppConnectionLdapCredentialsModel struct {
	Host          types.String `tfsdk:"host"`
	Port          types.Int32  `tfsdk:"port"`
	BindDn        types.String `tfsdk:"bind_dn"`
	BindPassword  types.String `tfsdk:"bind_password"`
	BaseDn        types.String `tfsdk:"base_dn"`
	TlsEnabled    types.Bool   `tfsdk:"tls_enabled"`
	TlsSkipVerify types.Bool   `tfsdk:"tls_skip_verify"`
	TlsCert       types.String `tfsdk:"tls_cert"`
	TlsKey        types.String `tfsdk:"tls_key"`
	TlsCa         types.String `tfsdk:"tls_ca"`
}

const AppConnectionLdapAuthMethodBindCredentials = "bind-credentials"

func NewAppConnectionLdapResource() resource.Resource {
	return &AppConnectionBaseResource{
		App:               infisical.AppConnectionAppLdap,
		AppConnectionName: "LDAP",
		ResourceTypeName:  "_app_connection_ldap",
		AllowedMethods:    []string{AppConnectionLdapAuthMethodBindCredentials},
		CredentialsAttributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Required:    true,
				Description: "The hostname or IP address of the LDAP server.",
			},
			"port": schema.Int32Attribute{
				Optional:    true,
				Computed:    true,
				Description: "The port number of the LDAP server.",
				Default:     int32default.StaticInt32(389),
			},
			"bind_dn": schema.StringAttribute{
				Required:    true,
				Description: "The Distinguished Name (DN) of the bind user for authentication.",
			},
			"bind_password": schema.StringAttribute{
				Required:    true,
				Description: "The password for the bind user.",
				Sensitive:   true,
			},
			"base_dn": schema.StringAttribute{
				Optional:    true,
				Description: "The base Distinguished Name (DN) for LDAP searches.",
			},
			"tls_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to use TLS when connecting to the LDAP server.",
				Default:     booldefault.StaticBool(false),
			},
			"tls_skip_verify": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to skip TLS certificate verification.",
				Default:     booldefault.StaticBool(false),
			},
			"tls_cert": schema.StringAttribute{
				Optional:    true,
				Description: "The TLS client certificate for authentication.",
				Sensitive:   true,
			},
			"tls_key": schema.StringAttribute{
				Optional:    true,
				Description: "The TLS client key for authentication.",
				Sensitive:   true,
			},
			"tls_ca": schema.StringAttribute{
				Optional:    true,
				Description: "The TLS certificate authority certificate.",
			},
		},
		ReadCredentialsForCreateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			credentialsConfig := make(map[string]any)

			var credentials AppConnectionLdapCredentialsModel
			diags := plan.Credentials.As(ctx, &credentials, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != AppConnectionLdapAuthMethodBindCredentials {
				diags.AddError(
					"Unable to create LDAP app connection",
					"Invalid method. Only bind-credentials method is supported",
				)
				return nil, diags
			}

			credentialsConfig["host"] = credentials.Host.ValueString()
			credentialsConfig["port"] = credentials.Port.ValueInt32()
			credentialsConfig["bindDn"] = credentials.BindDn.ValueString()
			credentialsConfig["bindPassword"] = credentials.BindPassword.ValueString()

			if !credentials.BaseDn.IsNull() {
				credentialsConfig["baseDn"] = credentials.BaseDn.ValueString()
			}

			credentialsConfig["tlsEnabled"] = credentials.TlsEnabled.ValueBool()
			credentialsConfig["tlsSkipVerify"] = credentials.TlsSkipVerify.ValueBool()

			if !credentials.TlsCert.IsNull() {
				credentialsConfig["tlsCert"] = credentials.TlsCert.ValueString()
			}
			if !credentials.TlsKey.IsNull() {
				credentialsConfig["tlsKey"] = credentials.TlsKey.ValueString()
			}
			if !credentials.TlsCa.IsNull() {
				credentialsConfig["tlsCa"] = credentials.TlsCa.ValueString()
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

			if plan.Method.ValueString() != AppConnectionLdapAuthMethodBindCredentials {
				diags.AddError(
					"Unable to update LDAP app connection",
					"Invalid method. Only bind-credentials method is supported",
				)
				return nil, diags
			}

			if credentialsFromState.Host.ValueString() != credentialsFromPlan.Host.ValueString() {
				credentialsConfig["host"] = credentialsFromPlan.Host.ValueString()
			}
			if credentialsFromState.Port.ValueInt32() != credentialsFromPlan.Port.ValueInt32() {
				credentialsConfig["port"] = credentialsFromPlan.Port.ValueInt32()
			}
			if credentialsFromState.BindDn.ValueString() != credentialsFromPlan.BindDn.ValueString() {
				credentialsConfig["bindDn"] = credentialsFromPlan.BindDn.ValueString()
			}
			if credentialsFromState.BindPassword.ValueString() != credentialsFromPlan.BindPassword.ValueString() {
				credentialsConfig["bindPassword"] = credentialsFromPlan.BindPassword.ValueString()
			}
			if credentialsFromState.BaseDn.ValueString() != credentialsFromPlan.BaseDn.ValueString() {
				if !credentialsFromPlan.BaseDn.IsNull() {
					credentialsConfig["baseDn"] = credentialsFromPlan.BaseDn.ValueString()
				} else {
					credentialsConfig["baseDn"] = ""
				}
			}
			if credentialsFromState.TlsEnabled.ValueBool() != credentialsFromPlan.TlsEnabled.ValueBool() {
				credentialsConfig["tlsEnabled"] = credentialsFromPlan.TlsEnabled.ValueBool()
			}
			if credentialsFromState.TlsSkipVerify.ValueBool() != credentialsFromPlan.TlsSkipVerify.ValueBool() {
				credentialsConfig["tlsSkipVerify"] = credentialsFromPlan.TlsSkipVerify.ValueBool()
			}
			if credentialsFromState.TlsCert.ValueString() != credentialsFromPlan.TlsCert.ValueString() {
				if !credentialsFromPlan.TlsCert.IsNull() {
					credentialsConfig["tlsCert"] = credentialsFromPlan.TlsCert.ValueString()
				} else {
					credentialsConfig["tlsCert"] = ""
				}
			}
			if credentialsFromState.TlsKey.ValueString() != credentialsFromPlan.TlsKey.ValueString() {
				if !credentialsFromPlan.TlsKey.IsNull() {
					credentialsConfig["tlsKey"] = credentialsFromPlan.TlsKey.ValueString()
				} else {
					credentialsConfig["tlsKey"] = ""
				}
			}
			if credentialsFromState.TlsCa.ValueString() != credentialsFromPlan.TlsCa.ValueString() {
				if !credentialsFromPlan.TlsCa.IsNull() {
					credentialsConfig["tlsCa"] = credentialsFromPlan.TlsCa.ValueString()
				} else {
					credentialsConfig["tlsCa"] = ""
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
					"host":            types.StringType,
					"port":            types.Int32Type,
					"bind_dn":         types.StringType,
					"bind_password":   types.StringType,
					"base_dn":         types.StringType,
					"tls_enabled":     types.BoolType,
					"tls_skip_verify": types.BoolType,
					"tls_cert":        types.StringType,
					"tls_key":         types.StringType,
					"tls_ca":          types.StringType,
				},
				map[string]attr.Value{
					"host":            credentials.Host,
					"port":            credentials.Port,
					"bind_dn":         credentials.BindDn,
					"bind_password":   credentials.BindPassword,
					"base_dn":         credentials.BaseDn,
					"tls_enabled":     credentials.TlsEnabled,
					"tls_skip_verify": credentials.TlsSkipVerify,
					"tls_cert":        credentials.TlsCert,
					"tls_key":         credentials.TlsKey,
					"tls_ca":          credentials.TlsCa,
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
