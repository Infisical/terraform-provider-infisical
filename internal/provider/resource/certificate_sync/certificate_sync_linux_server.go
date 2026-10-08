package resource

import (
	"context"
	infisical "terraform-provider-infisical/internal/client"
	customtypes "terraform-provider-infisical/internal/pkg/customtypes"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const linuxServerDefaultExportFormat = hostExportFormatPem

type CertificateSyncLinuxServerDestinationConfigModel struct {
	DestinationPath customtypes.TrimmedStringValue `tfsdk:"destination_path"`
	Host            customtypes.TrimmedStringValue `tfsdk:"host"`
	Port            types.Int64                    `tfsdk:"port"`
	SSHHostKeys     customtypes.TrimmedStringValue `tfsdk:"ssh_host_keys"`
}

type CertificateSyncLinuxServerSyncOptionsModel struct {
	CertificateSyncHostExportOptionsModel
	FileMode           customtypes.TrimmedStringValue `tfsdk:"file_mode"`
	PrivateKeyFileMode customtypes.TrimmedStringValue `tfsdk:"private_key_file_mode"`
	Owner              customtypes.TrimmedStringValue `tfsdk:"owner"`
	Group              customtypes.TrimmedStringValue `tfsdk:"group"`
}

var certificateSyncLinuxServerDestinationConfigAttrTypes = map[string]attr.Type{
	"destination_path": customtypes.TrimmedStringType{},
	"host":             customtypes.TrimmedStringType{},
	"port":             types.Int64Type,
	"ssh_host_keys":    customtypes.TrimmedStringType{},
}

var certificateSyncLinuxServerSyncOptionsAttrTypes = mergeHostAttributes(certificateSyncHostExportOptionsAttrTypes(), map[string]attr.Type{
	"file_mode":             customtypes.TrimmedStringType{},
	"private_key_file_mode": customtypes.TrimmedStringType{},
	"owner":                 customtypes.TrimmedStringType{},
	"group":                 customtypes.TrimmedStringType{},
})

func NewCertificateSyncLinuxServerResource() resource.Resource {
	return &CertificateSyncBaseResource{
		App:                    infisical.CertificateSyncAppLinuxServer,
		SyncName:               "Linux Server",
		ResourceTypeName:       "_certificate_sync_linux_server",
		AppConnection:          infisical.AppConnectionAppSSH,
		AdditionalConnections:  []infisical.AppConnectionApp{infisical.AppConnectionAppLdap},
		SupportsExportPassword: true,
		DestinationConfigAttributes: map[string]schema.Attribute{
			"destination_path": schema.StringAttribute{
				Required:    true,
				CustomType:  customtypes.TrimmedStringType{},
				Description: "The absolute directory on the server to write certificate files to (e.g. /etc/ssl/infisical).",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 4096),
				},
			},
			"host": schema.StringAttribute{
				Validators:  []validator.String{notBlank()},
				Optional:    true,
				CustomType:  customtypes.TrimmedStringType{},
				Description: "The hostname or IP address of the server to sync to. Required with an LDAP connection and not allowed with any other connection type.",
			},
			"port": schema.Int64Attribute{
				Optional:    true,
				Description: "The SSH port to reach the server on. Only valid with an LDAP connection.",
				Validators: []validator.Int64{
					int64validator.Between(1, 65535),
				},
			},
			"ssh_host_keys": schema.StringAttribute{
				Optional:    true,
				CustomType:  customtypes.TrimmedStringType{},
				Description: "The trusted SSH host keys of the server, as produced by `ssh-keyscan <host>`. The sync refuses to connect when the server presents a different key, and at least one RSA or ECDSA key must be included. Only valid with an LDAP connection.",
				Validators: []validator.String{
					notBlank(),
					stringvalidator.LengthAtMost(8192),
				},
			},
		},
		SyncOptionsAttributes: mergeHostAttributes(certificateSyncHostExportOptionsAttributes(linuxServerDefaultExportFormat), map[string]schema.Attribute{
			"file_mode": schema.StringAttribute{
				Validators:  []validator.String{notBlank()},
				Optional:    true,
				CustomType:  customtypes.TrimmedStringType{},
				Description: "The octal file mode applied to delivered certificate files (e.g. 0644).",
			},
			"private_key_file_mode": schema.StringAttribute{
				Validators:  []validator.String{notBlank()},
				Optional:    true,
				CustomType:  customtypes.TrimmedStringType{},
				Description: "The octal file mode applied to delivered private key files (e.g. 0600).",
			},
			"owner": schema.StringAttribute{
				Optional:    true,
				CustomType:  customtypes.TrimmedStringType{},
				Description: "The Linux user that should own the delivered files.",
				Validators: []validator.String{
					notBlank(),
					stringvalidator.LengthAtMost(32),
				},
			},
			"group": schema.StringAttribute{
				Optional:    true,
				CustomType:  customtypes.TrimmedStringType{},
				Description: "The Linux group that should own the delivered files.",
				Validators: []validator.String{
					notBlank(),
					stringvalidator.LengthAtMost(32),
				},
			},
		}),

		ValidateConfigFunc: func(ctx context.Context, config CertificateSyncBaseResourceModel, diags *diag.Diagnostics) {
			var syncOptions CertificateSyncLinuxServerSyncOptionsModel
			ok, d := decodeHostSyncOptions(ctx, config.SyncOptions, &syncOptions)
			diags.Append(d...)
			if !ok {
				return
			}
			validateCertificateSyncHostExportOptions(config, syncOptions.CertificateSyncHostExportOptionsModel, linuxServerDefaultExportFormat, diags)
		},

		ReadSyncOptionsFromPlan: func(ctx context.Context, plan CertificateSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			var syncOptions CertificateSyncLinuxServerSyncOptionsModel
			diags := plan.SyncOptions.As(ctx, &syncOptions, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			request := syncOptions.toRequest()
			setOptionalString(request, "fileMode", syncOptions.FileMode)
			setOptionalString(request, "privateKeyFileMode", syncOptions.PrivateKeyFileMode)
			setOptionalString(request, "owner", syncOptions.Owner)
			setOptionalString(request, "group", syncOptions.Group)
			return request, diags
		},

		ReadSyncOptionsFromApi: func(_ context.Context, certificateSync infisical.CertificateSync) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			values := certificateSyncHostExportOptionsFromApi(certificateSync.SyncOptions, linuxServerDefaultExportFormat, &diags)
			if diags.HasError() {
				return types.ObjectNull(certificateSyncLinuxServerSyncOptionsAttrTypes), diags
			}
			values["file_mode"] = optionalTrimmedStringFromMap(certificateSync.SyncOptions, "fileMode")
			values["private_key_file_mode"] = optionalTrimmedStringFromMap(certificateSync.SyncOptions, "privateKeyFileMode")
			values["owner"] = optionalTrimmedStringFromMap(certificateSync.SyncOptions, "owner")
			values["group"] = optionalTrimmedStringFromMap(certificateSync.SyncOptions, "group")

			return types.ObjectValue(certificateSyncLinuxServerSyncOptionsAttrTypes, values)
		},

		ReadDestinationConfigFromPlan: func(ctx context.Context, plan CertificateSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			var destinationConfig CertificateSyncLinuxServerDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &destinationConfig, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			request := map[string]interface{}{
				"destinationPath": destinationConfig.DestinationPath.ValueString(),
			}
			setOptionalString(request, "host", destinationConfig.Host)
			setOptionalInt64(request, "port", destinationConfig.Port)
			setOptionalString(request, "sshHostKeys", destinationConfig.SSHHostKeys)
			return request, diags
		},

		ReadDestinationConfigFromApi: func(_ context.Context, certificateSync infisical.CertificateSync) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			destinationPath := trimmedStringFromMap(certificateSync.DestinationConfig, "destinationPath", &diags)
			if diags.HasError() {
				return types.ObjectNull(certificateSyncLinuxServerDestinationConfigAttrTypes), diags
			}

			return types.ObjectValue(certificateSyncLinuxServerDestinationConfigAttrTypes, map[string]attr.Value{
				"destination_path": destinationPath,
				"host":             optionalTrimmedStringFromMap(certificateSync.DestinationConfig, "host"),
				"port":             optionalInt64FromMap(certificateSync.DestinationConfig, "port"),
				"ssh_host_keys":    optionalTrimmedStringFromMap(certificateSync.DestinationConfig, "sshHostKeys"),
			})
		},
	}
}
