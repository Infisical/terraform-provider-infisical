package resource

import (
	"context"
	"regexp"
	"strings"
	infisical "terraform-provider-infisical/internal/client"
	customtypes "terraform-provider-infisical/internal/pkg/customtypes"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type CertificateSyncPaloAltoNetworksDestinationConfigModel struct {
	Template      customtypes.TrimmedStringValue `tfsdk:"template"`
	PushToDevices types.Bool                     `tfsdk:"push_to_devices"`
}

type CertificateSyncPaloAltoNetworksSslTlsProfileDestinationConfigModel struct {
	Template                 customtypes.TrimmedStringValue `tfsdk:"template"`
	PushToDevices            types.Bool                     `tfsdk:"push_to_devices"`
	SslTlsServiceProfileName customtypes.TrimmedStringValue `tfsdk:"ssl_tls_service_profile_name"`
	SslTlsServiceProfileVsys customtypes.TrimmedStringValue `tfsdk:"ssl_tls_service_profile_vsys"`
}

type CertificateSyncPaloAltoNetworksSyncOptionsModel struct {
	CertificateNameSchema customtypes.TrimmedStringValue `tfsdk:"certificate_name_schema"`
	CanRemoveCertificates types.Bool                     `tfsdk:"can_remove_certificates"`
	IncludeRootCa         types.Bool                     `tfsdk:"include_root_ca"`
	PreserveItemOnRenewal types.Bool                     `tfsdk:"preserve_item_on_renewal"`
}

var certificateSyncPaloAltoNetworksDestinationConfigAttrTypes = map[string]attr.Type{
	"template":        customtypes.TrimmedStringType{},
	"push_to_devices": types.BoolType,
}

var certificateSyncPaloAltoNetworksSslTlsProfileDestinationConfigAttrTypes = map[string]attr.Type{
	"template":                     customtypes.TrimmedStringType{},
	"push_to_devices":              types.BoolType,
	"ssl_tls_service_profile_name": customtypes.TrimmedStringType{},
	"ssl_tls_service_profile_vsys": customtypes.TrimmedStringType{},
}

var certificateSyncPaloAltoNetworksSyncOptionsAttrTypes = map[string]attr.Type{
	"certificate_name_schema":  customtypes.TrimmedStringType{},
	"can_remove_certificates":  types.BoolType,
	"include_root_ca":          types.BoolType,
	"preserve_item_on_renewal": types.BoolType,
}

var (
	panOsObjectNamePattern              = regexp.MustCompile(`^[A-Za-z0-9 ._-]+$`)
	paloAltoNetworksCertificateNameRule = &certificateNameRule{pattern: regexp.MustCompile(`^[a-zA-Z0-9_-]{1,31}$`), requirement: "1-31 letters, digits, hyphens or underscores (use {{shortCertificateId}} rather than {{certificateId}} to stay within the limit)", placeholders: "{{shortCertificateId}}, {{commonName}}, {{applicationName}}"}
)

const paloAltoNetworksCertificateNameSchemaDescription = "The naming scheme for synced certificates, which must compile to 1-31 letters, digits, hyphens, or underscores. Available placeholders: {{shortCertificateId}}, {{commonName}} and {{applicationName}}."

func panOsObjectNameValidators() []validator.String {
	return []validator.String{
		notBlank(),
		stringvalidator.LengthAtMost(63),
		stringvalidator.RegexMatches(panOsObjectNamePattern, "must contain only letters, digits, spaces, periods, hyphens and underscores"),
	}
}

func paloAltoNetworksDestinationConfigAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"template": schema.StringAttribute{
			Optional:    true,
			CustomType:  customtypes.TrimmedStringType{},
			Description: "The Panorama template to sync certificates to. Required when the connection is a Panorama, and not allowed when it is a firewall.",
			Validators:  panOsObjectNameValidators(),
		},
		"push_to_devices": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Whether to push the template to the firewalls in every template stack that contains it after each sync. Applies only when the connection is a Panorama and `template` is set, and the template must then belong to a template stack. Infisical commits the changes on the connected firewall or Panorama either way. Defaults to `true`.",
			Default:     booldefault.StaticBool(true),
		},
	}
}

func paloAltoNetworksSyncOptionsAttributes(certificateNameSchemaDescription, canRemoveCertificatesDescription string) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"certificate_name_schema": schema.StringAttribute{
			Required:    true,
			CustomType:  customtypes.TrimmedStringType{},
			Description: certificateNameSchemaDescription,
		},
		"can_remove_certificates": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Description: canRemoveCertificatesDescription,
			Default:     booldefault.StaticBool(true),
		},
		"include_root_ca": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Whether to include the root CA certificate in the synced certificate chain. Defaults to `false`.",
			Default:     booldefault.StaticBool(false),
		},
		"preserve_item_on_renewal": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Whether a renewed certificate replaces its existing PAN-OS certificate instead of creating a new one. Defaults to `true`.",
			Default:     booldefault.StaticBool(true),
		},
	}
}

func readPaloAltoNetworksSyncOptionsFromPlan(ctx context.Context, plan CertificateSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
	var syncOptions CertificateSyncPaloAltoNetworksSyncOptionsModel
	diags := plan.SyncOptions.As(ctx, &syncOptions, objectAsOptions)
	if diags.HasError() {
		return nil, diags
	}

	return map[string]interface{}{
		"certificateNameSchema": syncOptions.CertificateNameSchema.ValueString(),
		"canRemoveCertificates": syncOptions.CanRemoveCertificates.ValueBool(),
		"includeRootCa":         syncOptions.IncludeRootCa.ValueBool(),
		"preserveItemOnRenewal": syncOptions.PreserveItemOnRenewal.ValueBool(),
	}, diags
}

func readPaloAltoNetworksSyncOptionsFromApi(_ context.Context, certificateSync infisical.CertificateSync) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	certificateNameSchema := trimmedStringFromMap(certificateSync.SyncOptions, "certificateNameSchema", &diags)
	if diags.HasError() {
		return types.ObjectNull(certificateSyncPaloAltoNetworksSyncOptionsAttrTypes), diags
	}

	return types.ObjectValue(certificateSyncPaloAltoNetworksSyncOptionsAttrTypes, map[string]attr.Value{
		"certificate_name_schema":  certificateNameSchema,
		"can_remove_certificates":  boolFromMap(certificateSync.SyncOptions, "canRemoveCertificates", true),
		"include_root_ca":          boolFromMap(certificateSync.SyncOptions, "includeRootCa", false),
		"preserve_item_on_renewal": boolFromMap(certificateSync.SyncOptions, "preserveItemOnRenewal", true),
	})
}

// validatePaloAltoNetworksNamePlaceholders mirrors the backend rule that a multi-certificate schema
// with placeholders must give each certificate its own name.
func validatePaloAltoNetworksNamePlaceholders(ctx context.Context, config CertificateSyncBaseResourceModel, diags *diag.Diagnostics) {
	schema, known, set := stringAttribute(ctx, config.SyncOptions, "certificate_name_schema")
	if !known || !set {
		return
	}
	if anyNamePlaceholderPattern.MatchString(schema) && !hasCertificateIdentifier(schema) && !strings.Contains(schema, "{{commonName}}") {
		diags.AddAttributeError(
			path.Root(attrSyncOptions).AtName("certificate_name_schema"),
			"Invalid certificate name schema",
			"It must include {{shortCertificateId}} or {{commonName}} so each certificate gets its own name, or use no placeholder for a single certificate.",
		)
	}
}

func NewCertificateSyncPaloAltoNetworksResource() resource.Resource {
	return &CertificateSyncBaseResource{
		App:                         infisical.CertificateSyncAppPaloAltoNetworks,
		SyncName:                    "Palo Alto Networks",
		DocsURL:                     "https://infisical.com/docs/documentation/platform/pki/applications/certificate-syncs/palo-alto-networks",
		ResourceTypeName:            "_certificate_sync_palo_alto_networks",
		CertificateNameRule:         paloAltoNetworksCertificateNameRule,
		AppConnection:               infisical.AppConnectionAppPaloAltoNetworks,
		DestinationConfigAttributes: paloAltoNetworksDestinationConfigAttributes(),
		SyncOptionsAttributes: paloAltoNetworksSyncOptionsAttributes(
			paloAltoNetworksCertificateNameSchemaDescription+" A schema with placeholders must include {{shortCertificateId}} or {{commonName}} so each certificate gets its own name. A schema with no placeholder can be linked to only one certificate.",
			"Whether Infisical should remove certificates from PAN-OS when their certificates leave this sync, for example when `certificate_filters` stops matching them. Destroying the sync leaves them in place. Defaults to `true`.",
		),

		ValidateConfigFunc: validatePaloAltoNetworksNamePlaceholders,

		ReadSyncOptionsFromPlan: readPaloAltoNetworksSyncOptionsFromPlan,
		ReadSyncOptionsFromApi:  readPaloAltoNetworksSyncOptionsFromApi,

		ReadDestinationConfigFromPlan: func(ctx context.Context, plan CertificateSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			var destinationConfig CertificateSyncPaloAltoNetworksDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &destinationConfig, objectAsOptions)
			if diags.HasError() {
				return nil, diags
			}

			request := map[string]interface{}{}
			setOptionalString(request, "template", destinationConfig.Template)
			setOptionalBool(request, "pushToDevices", destinationConfig.PushToDevices)
			return request, diags
		},

		ReadDestinationConfigFromApi: func(_ context.Context, certificateSync infisical.CertificateSync) (types.Object, diag.Diagnostics) {
			return types.ObjectValue(certificateSyncPaloAltoNetworksDestinationConfigAttrTypes, map[string]attr.Value{
				"template":        optionalTrimmedStringFromMap(certificateSync.DestinationConfig, "template"),
				"push_to_devices": boolFromMap(certificateSync.DestinationConfig, "pushToDevices", true),
			})
		},
	}
}

func NewCertificateSyncPaloAltoNetworksSslTlsProfileResource() resource.Resource {
	destinationConfigAttributes := paloAltoNetworksDestinationConfigAttributes()
	destinationConfigAttributes["ssl_tls_service_profile_name"] = schema.StringAttribute{
		Required:    true,
		CustomType:  customtypes.TrimmedStringType{},
		Description: "The name of the SSL/TLS service profile to bind the synced certificate to.",
		Validators:  panOsObjectNameValidators(),
	}
	destinationConfigAttributes["ssl_tls_service_profile_vsys"] = schema.StringAttribute{
		Optional:    true,
		CustomType:  customtypes.TrimmedStringType{},
		Description: "The virtual system that holds the SSL/TLS service profile, for example `vsys1`. Leave unset when the profile is in the shared scope.",
		Validators:  panOsObjectNameValidators(),
	}

	return &CertificateSyncBaseResource{
		App:                         infisical.CertificateSyncAppPaloAltoNetworksSslTlsProfile,
		SyncName:                    "Palo Alto Networks SSL/TLS Profile",
		DocsURL:                     "https://infisical.com/docs/documentation/platform/pki/applications/certificate-syncs/palo-alto-networks-ssl-tls-profile",
		ResourceTypeName:            "_certificate_sync_palo_alto_networks_ssl_tls_profile",
		CertificateNameRule:         paloAltoNetworksCertificateNameRule,
		AppConnection:               infisical.AppConnectionAppPaloAltoNetworks,
		DestinationConfigAttributes: destinationConfigAttributes,
		SyncOptionsAttributes: paloAltoNetworksSyncOptionsAttributes(
			paloAltoNetworksCertificateNameSchemaDescription,
			"Whether Infisical should remove the certificate from PAN-OS when it leaves this sync. Infisical refuses to empty the sync while this is `true`, because the SSL/TLS service profile would be left without a certificate; to change the profile's certificate, set `certificate_order_ids` to another certificate order instead. Destroying the sync leaves the certificate in place. Defaults to `true`.",
		),

		MaxCertificates: 1,

		ReadSyncOptionsFromPlan: readPaloAltoNetworksSyncOptionsFromPlan,
		ReadSyncOptionsFromApi:  readPaloAltoNetworksSyncOptionsFromApi,

		ReadDestinationConfigFromPlan: func(ctx context.Context, plan CertificateSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			var destinationConfig CertificateSyncPaloAltoNetworksSslTlsProfileDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &destinationConfig, objectAsOptions)
			if diags.HasError() {
				return nil, diags
			}

			request := map[string]interface{}{}
			setOptionalString(request, "template", destinationConfig.Template)
			setOptionalBool(request, "pushToDevices", destinationConfig.PushToDevices)
			setOptionalString(request, "sslTlsServiceProfileName", destinationConfig.SslTlsServiceProfileName)
			setOptionalString(request, "sslTlsServiceProfileVsys", destinationConfig.SslTlsServiceProfileVsys)
			return request, diags
		},

		ReadDestinationConfigFromApi: func(_ context.Context, certificateSync infisical.CertificateSync) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics
			sslTlsServiceProfileName := trimmedStringFromMap(certificateSync.DestinationConfig, "sslTlsServiceProfileName", &diags)
			if diags.HasError() {
				return types.ObjectNull(certificateSyncPaloAltoNetworksSslTlsProfileDestinationConfigAttrTypes), diags
			}

			return types.ObjectValue(certificateSyncPaloAltoNetworksSslTlsProfileDestinationConfigAttrTypes, map[string]attr.Value{
				"template":                     optionalTrimmedStringFromMap(certificateSync.DestinationConfig, "template"),
				"push_to_devices":              boolFromMap(certificateSync.DestinationConfig, "pushToDevices", true),
				"ssl_tls_service_profile_name": sslTlsServiceProfileName,
				"ssl_tls_service_profile_vsys": optionalTrimmedStringFromMap(certificateSync.DestinationConfig, "sslTlsServiceProfileVsys"),
			})
		},
	}
}
