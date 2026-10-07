package resource

import (
	"context"
	customtypes "terraform-provider-infisical/internal/pkg/customtypes"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	hostExportFormatPem    = "pem"
	hostExportFormatPkcs12 = "pkcs12"
	hostExportFormatJks    = "jks"

	hostPemCertificateExtensionPem = "pem"
	hostPemCertificateExtensionCrt = "crt"

	hostCommandMaxLength = 8192
)

type CertificateSyncHostExportOptionsModel struct {
	CertificateNameSchema   customtypes.TrimmedStringValue `tfsdk:"certificate_name_schema"`
	CanRemoveCertificates   types.Bool                     `tfsdk:"can_remove_certificates"`
	IncludeRootCa           types.Bool                     `tfsdk:"include_root_ca"`
	IncludePrivateKey       types.Bool                     `tfsdk:"include_private_key"`
	ExportFormat            types.String                   `tfsdk:"export_format"`
	PemCertificateExtension types.String                   `tfsdk:"pem_certificate_extension"`
	CombineCertificateChain types.Bool                     `tfsdk:"combine_certificate_chain"`
	KeystoreAlias           customtypes.TrimmedStringValue `tfsdk:"keystore_alias"`
	IncludeTruststore       types.Bool                     `tfsdk:"include_truststore"`
	HealthCheckCommand      customtypes.TrimmedStringValue `tfsdk:"health_check_command"`
	PostSyncCommand         customtypes.TrimmedStringValue `tfsdk:"post_sync_command"`
}

func certificateSyncHostExportOptionsAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"certificate_name_schema":   customtypes.TrimmedStringType{},
		"can_remove_certificates":   types.BoolType,
		"include_root_ca":           types.BoolType,
		"include_private_key":       types.BoolType,
		"export_format":             types.StringType,
		"pem_certificate_extension": types.StringType,
		"combine_certificate_chain": types.BoolType,
		"keystore_alias":            customtypes.TrimmedStringType{},
		"include_truststore":        types.BoolType,
		"health_check_command":      customtypes.TrimmedStringType{},
		"post_sync_command":         customtypes.TrimmedStringType{},
	}
}

func certificateSyncHostExportOptionsAttributes(defaultExportFormat string) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"certificate_name_schema": schema.StringAttribute{
			Required:    true,
			CustomType:  customtypes.TrimmedStringType{},
			Description: "The file base name for synced certificates. It must resolve to a single file name of 1-200 characters using only letters, digits, dots (.), dashes (-), and underscores (_). Available placeholders: {{certificateId}}, {{shortCertificateId}}, {{profileId}}, {{applicationId}}, {{applicationName}}, {{commonName}}. A schema with no placeholder can be linked to only one certificate.",
		},
		"can_remove_certificates": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Whether Infisical should delete certificate files from the server when the certificates are no longer managed in Infisical. Defaults to `false`.",
			Default:     booldefault.StaticBool(false),
		},
		"include_root_ca": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Whether to include the root CA certificate in the delivered certificate chain. Defaults to `false`.",
			Default:     booldefault.StaticBool(false),
		},
		"include_private_key": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Whether to deliver the certificate's private key alongside the certificate. Defaults to `true`.",
			Default:     booldefault.StaticBool(true),
		},
		"export_format": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "The file format to deliver certificates in. Supported values: `pem`, `pkcs12`, `jks`. Defaults to `" + defaultExportFormat + "`.",
			Default:     stringdefault.StaticString(defaultExportFormat),
			Validators: []validator.String{
				stringvalidator.OneOf(hostExportFormatPem, hostExportFormatPkcs12, hostExportFormatJks),
			},
		},
		"pem_certificate_extension": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "The file extension of PEM certificate and chain files. Supported values: `pem`, `crt`. Defaults to `pem`.",
			Default:     stringdefault.StaticString(hostPemCertificateExtensionPem),
			Validators: []validator.String{
				stringvalidator.OneOf(hostPemCertificateExtensionPem, hostPemCertificateExtensionCrt),
			},
		},
		"combine_certificate_chain": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Whether to write the leaf certificate followed by the chain into a single PEM file instead of a separate chain file. Defaults to `false`.",
			Default:     booldefault.StaticBool(false),
		},
		"keystore_alias": schema.StringAttribute{
			Optional:    true,
			CustomType:  customtypes.TrimmedStringType{},
			Description: "The alias of the private key entry in a PKCS#12 or JKS keystore, using only letters, digits, dots (.), dashes (-), and underscores (_). Only valid when `export_format` is `pkcs12` or `jks`, and defaults to the certificate's file base name.",
			Validators: []validator.String{
				notBlank(),
				stringvalidator.LengthBetween(1, 128),
			},
		},
		"include_truststore": schema.BoolAttribute{
			Optional:    true,
			Description: "Whether to also deliver a `<name>.truststore.jks` file holding the chain and root CA as trusted certificates. Only valid when `export_format` is `jks`.",
		},
		"health_check_command": schema.StringAttribute{
			Optional:    true,
			CustomType:  customtypes.TrimmedStringType{},
			Description: "A command run on the server to check the health of the delivered certificates.",
			Validators: []validator.String{
				notBlank(),
				stringvalidator.LengthAtMost(hostCommandMaxLength),
			},
		},
		"post_sync_command": schema.StringAttribute{
			Optional:    true,
			CustomType:  customtypes.TrimmedStringType{},
			Description: "A command run on the server after certificates are delivered, for example to reload a service.",
			Validators: []validator.String{
				notBlank(),
				stringvalidator.LengthAtMost(hostCommandMaxLength),
			},
		},
	}
}

func mergeHostAttributes[T any](maps ...map[string]T) map[string]T {
	merged := map[string]T{}
	for _, m := range maps {
		for k, v := range m {
			merged[k] = v
		}
	}
	return merged
}

func hostCommandForRequest(value customtypes.TrimmedStringValue) interface{} {
	if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
		return nil
	}
	return value.ValueString()
}

func (o CertificateSyncHostExportOptionsModel) toRequest() map[string]interface{} {
	syncOptions := map[string]interface{}{
		"certificateNameSchema":   o.CertificateNameSchema.ValueString(),
		"canRemoveCertificates":   o.CanRemoveCertificates.ValueBool(),
		"includeRootCa":           o.IncludeRootCa.ValueBool(),
		"includePrivateKey":       o.IncludePrivateKey.ValueBool(),
		"exportFormat":            o.ExportFormat.ValueString(),
		"pemCertificateExtension": o.PemCertificateExtension.ValueString(),
		"combineCertificateChain": o.CombineCertificateChain.ValueBool(),
		// The API keeps the stored command when the key is omitted, so null is sent to clear it.
		"healthCheckCommand": hostCommandForRequest(o.HealthCheckCommand),
		"postSyncCommand":    hostCommandForRequest(o.PostSyncCommand),
	}
	setOptionalString(syncOptions, "keystoreAlias", o.KeystoreAlias)
	setOptionalBool(syncOptions, "includeTruststore", o.IncludeTruststore)
	return syncOptions
}

func certificateSyncHostExportOptionsFromApi(m map[string]interface{}, defaultExportFormat string, diags *diag.Diagnostics) map[string]attr.Value {
	certificateNameSchema := trimmedStringFromMap(m, "certificateNameSchema", diags)

	exportFormat := optionalStringFromMap(m, "exportFormat")
	if exportFormat.IsNull() {
		exportFormat = types.StringValue(defaultExportFormat)
	}
	pemCertificateExtension := optionalStringFromMap(m, "pemCertificateExtension")
	if pemCertificateExtension.IsNull() {
		pemCertificateExtension = types.StringValue(hostPemCertificateExtensionPem)
	}

	return map[string]attr.Value{
		"certificate_name_schema":   certificateNameSchema,
		"can_remove_certificates":   boolFromMap(m, "canRemoveCertificates", false),
		"include_root_ca":           boolFromMap(m, "includeRootCa", false),
		"include_private_key":       boolFromMap(m, "includePrivateKey", true),
		"export_format":             exportFormat,
		"pem_certificate_extension": pemCertificateExtension,
		"combine_certificate_chain": boolFromMap(m, "combineCertificateChain", false),
		"keystore_alias":            optionalTrimmedStringFromMap(m, "keystoreAlias"),
		"include_truststore":        optionalBoolFromMap(m, "includeTruststore"),
		"health_check_command":      optionalTrimmedStringFromMap(m, "healthCheckCommand"),
		"post_sync_command":         optionalTrimmedStringFromMap(m, "postSyncCommand"),
	}
}

func isHostKeystoreExportFormat(format string) bool {
	return format == hostExportFormatPkcs12 || format == hostExportFormatJks
}

func validateCertificateSyncHostExportOptions(config CertificateSyncBaseResourceModel, options CertificateSyncHostExportOptionsModel, defaultExportFormat string, diags *diag.Diagnostics) {
	if options.ExportFormat.IsUnknown() {
		return
	}
	exportFormat := defaultExportFormat
	if !options.ExportFormat.IsNull() {
		exportFormat = options.ExportFormat.ValueString()
	}
	exportFormatPath := path.Root(attrSyncOptions).AtName("export_format")

	if isHostKeystoreExportFormat(exportFormat) &&
		!config.ExportPassword.IsUnknown() && !config.ExportPasswordWO.IsUnknown() &&
		config.ExportPassword.IsNull() && config.ExportPasswordWO.IsNull() {
		diags.AddAttributeError(
			exportFormatPath,
			"Missing export password",
			"`export_password` or `export_password_wo` must be set when `export_format` is `pkcs12` or `jks`.",
		)
	}

	if !options.KeystoreAlias.IsNull() && !options.KeystoreAlias.IsUnknown() && !isHostKeystoreExportFormat(exportFormat) {
		diags.AddAttributeError(
			path.Root(attrSyncOptions).AtName("keystore_alias"),
			"Invalid keystore alias",
			"`keystore_alias` only applies when `export_format` is `pkcs12` or `jks`. Remove it or change the export format.",
		)
	}

	if !options.IncludeTruststore.IsUnknown() && options.IncludeTruststore.ValueBool() && exportFormat != hostExportFormatJks {
		diags.AddAttributeError(
			path.Root(attrSyncOptions).AtName("include_truststore"),
			"Invalid include_truststore",
			"`include_truststore` only applies when `export_format` is `jks`. Remove it or change the export format.",
		)
	}
}

func decodeHostSyncOptions(ctx context.Context, syncOptions types.Object, target interface{}) (bool, diag.Diagnostics) {
	if syncOptions.IsNull() || syncOptions.IsUnknown() {
		return false, nil
	}
	diags := syncOptions.As(ctx, target, objectAsOptions)
	return !diags.HasError(), diags
}
