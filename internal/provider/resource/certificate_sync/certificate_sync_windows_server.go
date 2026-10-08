package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"
	customtypes "terraform-provider-infisical/internal/pkg/customtypes"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const (
	windowsServerDefaultExportFormat = hostExportFormatPkcs12

	windowsFileAccessRead        = "read"
	windowsFileAccessModify      = "modify"
	windowsFileAccessFullControl = "full-control"
)

type CertificateSyncWindowsServerDestinationConfigModel struct {
	DestinationPath       customtypes.TrimmedStringValue `tfsdk:"destination_path"`
	Host                  customtypes.TrimmedStringValue `tfsdk:"host"`
	Port                  types.Int64                    `tfsdk:"port"`
	SSLEnabled            types.Bool                     `tfsdk:"ssl_enabled"`
	SSLRejectUnauthorized types.Bool                     `tfsdk:"ssl_reject_unauthorized"`
	SSLCertificate        customtypes.TrimmedStringValue `tfsdk:"ssl_certificate"`
}

type CertificateSyncWindowsServerFileAccessRuleModel struct {
	Identity customtypes.TrimmedStringValue `tfsdk:"identity"`
	Access   types.String                   `tfsdk:"access"`
}

type CertificateSyncWindowsServerSyncOptionsModel struct {
	CertificateSyncHostExportOptionsModel
	FileAccessRules types.List `tfsdk:"file_access_rules"`
}

var certificateSyncWindowsServerDestinationConfigAttrTypes = map[string]attr.Type{
	"destination_path":        customtypes.TrimmedStringType{},
	"host":                    customtypes.TrimmedStringType{},
	"port":                    types.Int64Type,
	"ssl_enabled":             types.BoolType,
	"ssl_reject_unauthorized": types.BoolType,
	"ssl_certificate":         customtypes.TrimmedStringType{},
}

var certificateSyncWindowsServerFileAccessRuleAttrTypes = map[string]attr.Type{
	"identity": customtypes.TrimmedStringType{},
	"access":   types.StringType,
}

var certificateSyncWindowsServerFileAccessRuleObjectType = types.ObjectType{AttrTypes: certificateSyncWindowsServerFileAccessRuleAttrTypes}

var certificateSyncWindowsServerSyncOptionsAttrTypes = mergeHostAttributes(certificateSyncHostExportOptionsAttrTypes(), map[string]attr.Type{
	"file_access_rules": types.ListType{ElemType: certificateSyncWindowsServerFileAccessRuleObjectType},
})

func windowsServerFileAccessRulesFromApi(m map[string]interface{}) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	raw, ok := m["fileAccessRules"].([]interface{})
	if !ok {
		return types.ListNull(certificateSyncWindowsServerFileAccessRuleObjectType), diags
	}

	rules := make([]attr.Value, 0, len(raw))
	for i, item := range raw {
		rule, ok := item.(map[string]interface{})
		if !ok {
			diags.AddError("Invalid fileAccessRules type", fmt.Sprintf("Expected 'fileAccessRules[%d]' to be an object but got something else", i))
			return types.ListNull(certificateSyncWindowsServerFileAccessRuleObjectType), diags
		}
		identity := trimmedStringFromMap(rule, "identity", &diags)
		access := stringFromMap(rule, "access", &diags)
		if diags.HasError() {
			return types.ListNull(certificateSyncWindowsServerFileAccessRuleObjectType), diags
		}
		value, d := types.ObjectValue(certificateSyncWindowsServerFileAccessRuleAttrTypes, map[string]attr.Value{
			"identity": identity,
			"access":   access,
		})
		diags.Append(d...)
		if diags.HasError() {
			return types.ListNull(certificateSyncWindowsServerFileAccessRuleObjectType), diags
		}
		rules = append(rules, value)
	}

	list, d := types.ListValue(certificateSyncWindowsServerFileAccessRuleObjectType, rules)
	diags.Append(d...)
	return list, diags
}

func NewCertificateSyncWindowsServerResource() resource.Resource {
	return &CertificateSyncBaseResource{
		App:                    infisical.CertificateSyncAppWindowsServer,
		SyncName:               "Windows Server",
		ResourceTypeName:       "_certificate_sync_windows_server",
		AppConnection:          infisical.AppConnectionAppWinRM,
		AdditionalConnections:  []infisical.AppConnectionApp{infisical.AppConnectionAppLdap},
		SupportsExportPassword: true,
		DestinationConfigAttributes: map[string]schema.Attribute{
			"destination_path": schema.StringAttribute{
				Required:    true,
				CustomType:  customtypes.TrimmedStringType{},
				Description: "The absolute local drive directory on the server to write certificate files to (e.g. C:\\certs), where UNC paths are not supported.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 4096),
				},
			},
			"host": schema.StringAttribute{
				Validators:  []validator.String{notBlank()},
				Optional:    true,
				CustomType:  customtypes.TrimmedStringType{},
				Description: "The hostname or IP address of the server to sync to. Required with an LDAP connection and not allowed with any other connection type. The LDAP connection must have a gateway.",
			},
			"port": schema.Int64Attribute{
				Optional:    true,
				Description: "The WinRM port to reach the server on. Only valid with an LDAP connection.",
				Validators: []validator.Int64{
					int64validator.Between(1, 65535),
				},
			},
			"ssl_enabled": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether to reach the server over HTTPS WinRM instead of HTTP with NTLM message encryption. Only valid with an LDAP connection.",
			},
			"ssl_reject_unauthorized": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether to verify the server's WinRM certificate when using HTTPS. Only valid with an LDAP connection.",
			},
			"ssl_certificate": schema.StringAttribute{
				Optional:    true,
				CustomType:  customtypes.TrimmedStringType{},
				Description: "The PEM-encoded CA certificate used to verify a self-signed WinRM HTTPS listener. Only valid with an LDAP connection.",
				Validators: []validator.String{
					notBlank(),
					stringvalidator.LengthAtMost(8192),
				},
			},
		},
		SyncOptionsAttributes: mergeHostAttributes(certificateSyncHostExportOptionsAttributes(windowsServerDefaultExportFormat), map[string]schema.Attribute{
			"file_access_rules": schema.ListNestedAttribute{
				Optional:    true,
				Description: "Access rules granted on the delivered files to Windows users or groups, up to 20.",
				Validators: []validator.List{
					listvalidator.SizeAtMost(20),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"identity": schema.StringAttribute{
							Required:    true,
							CustomType:  customtypes.TrimmedStringType{},
							Description: "The Windows user or group to grant access to (e.g. DOMAIN\\svc-account or BUILTIN\\Administrators).",
							Validators: []validator.String{
								notBlank(),
								stringvalidator.LengthBetween(1, 256),
							},
						},
						"access": schema.StringAttribute{
							Required:    true,
							Description: "The access level to grant. Supported values: `read`, `modify`, `full-control`.",
							Validators: []validator.String{
								stringvalidator.OneOf(windowsFileAccessRead, windowsFileAccessModify, windowsFileAccessFullControl),
							},
						},
					},
				},
			},
		}),

		ValidateConfigFunc: func(ctx context.Context, config CertificateSyncBaseResourceModel, diags *diag.Diagnostics) {
			var syncOptions CertificateSyncWindowsServerSyncOptionsModel
			ok, d := decodeHostSyncOptions(ctx, config.SyncOptions, &syncOptions)
			diags.Append(d...)
			if !ok {
				return
			}
			validateCertificateSyncHostExportOptions(config, syncOptions.CertificateSyncHostExportOptionsModel, windowsServerDefaultExportFormat, diags)
		},

		ReadSyncOptionsFromPlan: func(ctx context.Context, plan CertificateSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			var syncOptions CertificateSyncWindowsServerSyncOptionsModel
			diags := plan.SyncOptions.As(ctx, &syncOptions, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			request := syncOptions.toRequest()

			if !syncOptions.FileAccessRules.IsNull() && !syncOptions.FileAccessRules.IsUnknown() {
				var rules []CertificateSyncWindowsServerFileAccessRuleModel
				diags.Append(syncOptions.FileAccessRules.ElementsAs(ctx, &rules, false)...)
				if diags.HasError() {
					return nil, diags
				}
				fileAccessRules := make([]map[string]interface{}, 0, len(rules))
				for _, rule := range rules {
					fileAccessRules = append(fileAccessRules, map[string]interface{}{
						"identity": rule.Identity.ValueString(),
						"access":   rule.Access.ValueString(),
					})
				}
				request["fileAccessRules"] = fileAccessRules
			}

			return request, diags
		},

		ReadSyncOptionsFromApi: func(_ context.Context, certificateSync infisical.CertificateSync) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			values := certificateSyncHostExportOptionsFromApi(certificateSync.SyncOptions, windowsServerDefaultExportFormat, &diags)
			if diags.HasError() {
				return types.ObjectNull(certificateSyncWindowsServerSyncOptionsAttrTypes), diags
			}

			fileAccessRules, d := windowsServerFileAccessRulesFromApi(certificateSync.SyncOptions)
			diags.Append(d...)
			if diags.HasError() {
				return types.ObjectNull(certificateSyncWindowsServerSyncOptionsAttrTypes), diags
			}
			values["file_access_rules"] = fileAccessRules

			obj, d := types.ObjectValue(certificateSyncWindowsServerSyncOptionsAttrTypes, values)
			diags.Append(d...)
			return obj, diags
		},

		ReadDestinationConfigFromPlan: func(ctx context.Context, plan CertificateSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			var destinationConfig CertificateSyncWindowsServerDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &destinationConfig, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			request := map[string]interface{}{
				"destinationPath": destinationConfig.DestinationPath.ValueString(),
			}
			setOptionalString(request, "host", destinationConfig.Host)
			setOptionalInt64(request, "port", destinationConfig.Port)
			setOptionalBool(request, "sslEnabled", destinationConfig.SSLEnabled)
			setOptionalBool(request, "sslRejectUnauthorized", destinationConfig.SSLRejectUnauthorized)
			setOptionalString(request, "sslCertificate", destinationConfig.SSLCertificate)
			return request, diags
		},

		ReadDestinationConfigFromApi: func(_ context.Context, certificateSync infisical.CertificateSync) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			destinationPath := trimmedStringFromMap(certificateSync.DestinationConfig, "destinationPath", &diags)
			if diags.HasError() {
				return types.ObjectNull(certificateSyncWindowsServerDestinationConfigAttrTypes), diags
			}

			return types.ObjectValue(certificateSyncWindowsServerDestinationConfigAttrTypes, map[string]attr.Value{
				"destination_path":        destinationPath,
				"host":                    optionalTrimmedStringFromMap(certificateSync.DestinationConfig, "host"),
				"port":                    optionalInt64FromMap(certificateSync.DestinationConfig, "port"),
				"ssl_enabled":             optionalBoolFromMap(certificateSync.DestinationConfig, "sslEnabled"),
				"ssl_reject_unauthorized": optionalBoolFromMap(certificateSync.DestinationConfig, "sslRejectUnauthorized"),
				"ssl_certificate":         optionalTrimmedStringFromMap(certificateSync.DestinationConfig, "sslCertificate"),
			})
		},
	}
}
