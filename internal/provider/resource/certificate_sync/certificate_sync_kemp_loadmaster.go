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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const kempLoadMasterDefaultCaCertificateNameSchema = "Infisical-ca-{{fingerprint}}"

type CertificateSyncKempLoadMasterDestinationConfigModel struct {
	VirtualServiceID customtypes.TrimmedStringValue `tfsdk:"virtual_service_id"`
}

type CertificateSyncKempLoadMasterSyncOptionsModel struct {
	CertificateNameSchema   customtypes.TrimmedStringValue `tfsdk:"certificate_name_schema"`
	CaCertificateNameSchema customtypes.TrimmedStringValue `tfsdk:"ca_certificate_name_schema"`
	CanRemoveCertificates   types.Bool                     `tfsdk:"can_remove_certificates"`
	IncludeRootCa           types.Bool                     `tfsdk:"include_root_ca"`
	PreserveItemOnRenewal   types.Bool                     `tfsdk:"preserve_item_on_renewal"`
}

var certificateSyncKempLoadMasterDestinationConfigAttrTypes = map[string]attr.Type{
	"virtual_service_id": customtypes.TrimmedStringType{},
}

var certificateSyncKempLoadMasterSyncOptionsAttrTypes = map[string]attr.Type{
	"certificate_name_schema":    customtypes.TrimmedStringType{},
	"ca_certificate_name_schema": customtypes.TrimmedStringType{},
	"can_remove_certificates":    types.BoolType,
	"include_root_ca":            types.BoolType,
	"preserve_item_on_renewal":   types.BoolType,
}

func kempIsVirtualServiceIndex(value string) bool {
	if value == "" {
		return false
	}
	for _, c := range value {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func validateKempLoadMasterDestinationConfig(ctx context.Context, config CertificateSyncBaseResourceModel, diags *diag.Diagnostics) {
	if config.DestinationConfig.IsNull() || config.DestinationConfig.IsUnknown() {
		return
	}

	var destinationConfig CertificateSyncKempLoadMasterDestinationConfigModel
	diags.Append(config.DestinationConfig.As(ctx, &destinationConfig, objectAsOptions)...)
	if diags.HasError() {
		return
	}

	virtualServiceID := destinationConfig.VirtualServiceID
	if virtualServiceID.IsNull() || virtualServiceID.IsUnknown() {
		return
	}
	if !kempIsVirtualServiceIndex(strings.TrimSpace(virtualServiceID.ValueString())) {
		diags.AddAttributeError(path.Root(attrDestinationConfig).AtName("virtual_service_id"), "Invalid virtual service ID",
			"virtual_service_id must be the numeric Virtual Service index shown on the LoadMaster.")
	}
}

func NewCertificateSyncKempLoadMasterResource() resource.Resource {
	return &CertificateSyncBaseResource{
		App:                 infisical.CertificateSyncAppKempLoadMaster,
		SyncName:            "Kemp LoadMaster",
		ResourceTypeName:    "_certificate_sync_kemp_loadmaster",
		CertificateNameRule: &certificateNameRule{pattern: regexp.MustCompile(`^[a-zA-Z0-9._-]{1,251}$`), requireIdentifier: true, requirement: "1-251 letters, digits, periods, hyphens or underscores"},
		AppConnection:       infisical.AppConnectionAppKempLoadMaster,
		DestinationConfigAttributes: map[string]schema.Attribute{
			"virtual_service_id": schema.StringAttribute{
				Optional:    true,
				CustomType:  customtypes.TrimmedStringType{},
				Description: "The numeric index of the Virtual Service to bind synced certificates to, as shown on the LoadMaster. Leave unset to sync certificates without binding them to a Virtual Service.",
				Validators:  []validator.String{stringvalidator.LengthAtMost(10)},
			},
		},
		SyncOptionsAttributes: map[string]schema.Attribute{
			"certificate_name_schema": schema.StringAttribute{
				Required:    true,
				CustomType:  customtypes.TrimmedStringType{},
				Description: "The naming scheme for synced certificates. Must include the {{certificateId}} or {{shortCertificateId}} placeholder and compile to 1-251 alphanumeric characters, hyphens, underscores, or periods. Available placeholders: {{certificateId}}, {{shortCertificateId}}, {{profileId}}, {{applicationId}}, {{applicationName}}, {{commonName}}.",
			},
			"ca_certificate_name_schema": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				CustomType:  customtypes.TrimmedStringType{},
				Description: "The naming scheme for synced CA certificates, which must compile to 1-251 alphanumeric characters, hyphens, underscores, or periods. Available placeholders: {{fingerprint}} (recommended so each CA gets a unique name) and {{commonName}}. Defaults to `Infisical-ca-{{fingerprint}}`.",
				Default:     stringdefault.StaticString(kempLoadMasterDefaultCaCertificateNameSchema),
			},
			"can_remove_certificates": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether Infisical should remove certificates from Kemp LoadMaster when they are no longer managed in Infisical. Defaults to `true`.",
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
				Description: "Whether a renewed certificate replaces its existing Kemp LoadMaster certificate instead of creating a new one. Defaults to `true`.",
				Default:     booldefault.StaticBool(true),
			},
		},

		ValidateConfigFunc: func(ctx context.Context, config CertificateSyncBaseResourceModel, diags *diag.Diagnostics) {
			validateKempLoadMasterDestinationConfig(ctx, config, diags)
			validateKempLoadMasterCaNameSchema(ctx, config, diags)
		},

		ReadSyncOptionsFromPlan: func(ctx context.Context, plan CertificateSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			var syncOptions CertificateSyncKempLoadMasterSyncOptionsModel
			diags := plan.SyncOptions.As(ctx, &syncOptions, objectAsOptions)
			if diags.HasError() {
				return nil, diags
			}

			return map[string]interface{}{
				"certificateNameSchema":   syncOptions.CertificateNameSchema.ValueString(),
				"caCertificateNameSchema": syncOptions.CaCertificateNameSchema.ValueString(),
				"canRemoveCertificates":   syncOptions.CanRemoveCertificates.ValueBool(),
				"includeRootCa":           syncOptions.IncludeRootCa.ValueBool(),
				"preserveItemOnRenewal":   syncOptions.PreserveItemOnRenewal.ValueBool(),
			}, diags
		},

		ReadSyncOptionsFromApi: func(_ context.Context, certificateSync infisical.CertificateSync) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			certificateNameSchema := trimmedStringFromMap(certificateSync.SyncOptions, "certificateNameSchema", &diags)
			if diags.HasError() {
				return types.ObjectNull(certificateSyncKempLoadMasterSyncOptionsAttrTypes), diags
			}

			caCertificateNameSchema := optionalTrimmedStringFromMap(certificateSync.SyncOptions, "caCertificateNameSchema")
			if caCertificateNameSchema.IsNull() {
				caCertificateNameSchema = customtypes.NewTrimmedStringValue(kempLoadMasterDefaultCaCertificateNameSchema)
			}

			return types.ObjectValue(certificateSyncKempLoadMasterSyncOptionsAttrTypes, map[string]attr.Value{
				"certificate_name_schema":    certificateNameSchema,
				"ca_certificate_name_schema": caCertificateNameSchema,
				"can_remove_certificates":    boolFromMap(certificateSync.SyncOptions, "canRemoveCertificates", true),
				"include_root_ca":            boolFromMap(certificateSync.SyncOptions, "includeRootCa", false),
				"preserve_item_on_renewal":   boolFromMap(certificateSync.SyncOptions, "preserveItemOnRenewal", true),
			})
		},

		ReadDestinationConfigFromPlan: func(ctx context.Context, plan CertificateSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			var destinationConfig CertificateSyncKempLoadMasterDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &destinationConfig, objectAsOptions)
			if diags.HasError() {
				return nil, diags
			}

			request := map[string]interface{}{}
			setOptionalString(request, "virtualServiceId", destinationConfig.VirtualServiceID)
			return request, diags
		},

		ReadDestinationConfigFromApi: func(_ context.Context, certificateSync infisical.CertificateSync) (types.Object, diag.Diagnostics) {
			return types.ObjectValue(certificateSyncKempLoadMasterDestinationConfigAttrTypes, map[string]attr.Value{
				"virtual_service_id": optionalTrimmedStringFromMap(certificateSync.DestinationConfig, "virtualServiceId"),
			})
		},
	}
}

var kempLoadMasterNamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,251}$`)

// validateKempLoadMasterCaNameSchema mirrors the backend check, which fills {{fingerprint}} and
// {{commonName}} with stand-in values before matching the Kemp naming rule.
func validateKempLoadMasterCaNameSchema(ctx context.Context, config CertificateSyncBaseResourceModel, diags *diag.Diagnostics) {
	schema, known, set := stringAttribute(ctx, config.SyncOptions, "ca_certificate_name_schema")
	if !known || !set {
		return
	}
	compiled := strings.ReplaceAll(strings.TrimSpace(schema), "{{fingerprint}}", strings.Repeat("0", 24))
	compiled = strings.ReplaceAll(compiled, "{{commonName}}", "common-name")
	if !kempLoadMasterNamePattern.MatchString(compiled) {
		diags.AddAttributeError(path.Root(attrSyncOptions).AtName("ca_certificate_name_schema"), "Invalid CA certificate name schema",
			"With placeholders filled in it must be 1-251 letters, digits, periods, hyphens or underscores. Available placeholders: {{fingerprint}}, {{commonName}}.")
	}
}
