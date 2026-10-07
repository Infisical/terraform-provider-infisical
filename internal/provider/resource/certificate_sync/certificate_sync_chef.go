package resource

import (
	"context"
	infisical "terraform-provider-infisical/internal/client"
	customtypes "terraform-provider-infisical/internal/pkg/customtypes"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	chefDefaultFieldCertificate      = "certificate"
	chefDefaultFieldPrivateKey       = "private_key"
	chefDefaultFieldCertificateChain = "certificate_chain"
	chefDefaultFieldCaCertificate    = "ca_certificate"
)

type CertificateSyncChefDestinationConfigModel struct {
	DataBagName customtypes.TrimmedStringValue `tfsdk:"data_bag_name"`
}

type CertificateSyncChefFieldMappingsModel struct {
	Certificate      types.String `tfsdk:"certificate"`
	PrivateKey       types.String `tfsdk:"private_key"`
	CertificateChain types.String `tfsdk:"certificate_chain"`
	CaCertificate    types.String `tfsdk:"ca_certificate"`
}

type CertificateSyncChefSyncOptionsModel struct {
	CertificateNameSchema      customtypes.TrimmedStringValue `tfsdk:"certificate_name_schema"`
	CanRemoveCertificates      types.Bool                     `tfsdk:"can_remove_certificates"`
	IncludeRootCa              types.Bool                     `tfsdk:"include_root_ca"`
	PreserveItemOnRenewal      types.Bool                     `tfsdk:"preserve_item_on_renewal"`
	UpdateExistingCertificates types.Bool                     `tfsdk:"update_existing_certificates"`
	FieldMappings              types.Object                   `tfsdk:"field_mappings"`
}

var certificateSyncChefDestinationConfigAttrTypes = map[string]attr.Type{
	"data_bag_name": customtypes.TrimmedStringType{},
}

var certificateSyncChefFieldMappingsAttrTypes = map[string]attr.Type{
	"certificate":       types.StringType,
	"private_key":       types.StringType,
	"certificate_chain": types.StringType,
	"ca_certificate":    types.StringType,
}

var certificateSyncChefSyncOptionsAttrTypes = map[string]attr.Type{
	"certificate_name_schema":      customtypes.TrimmedStringType{},
	"can_remove_certificates":      types.BoolType,
	"include_root_ca":              types.BoolType,
	"preserve_item_on_renewal":     types.BoolType,
	"update_existing_certificates": types.BoolType,
	"field_mappings":               types.ObjectType{AttrTypes: certificateSyncChefFieldMappingsAttrTypes},
}

func chefFieldMappingAttribute(description, def string) schema.StringAttribute {
	return schema.StringAttribute{
		Optional:    true,
		Computed:    true,
		Description: description,
		Default:     stringdefault.StaticString(def),
		Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
	}
}

func chefFieldMappingFromMap(m map[string]interface{}, key, def string) types.String {
	if value, ok := m[key].(string); ok && value != "" {
		return types.StringValue(value)
	}
	return types.StringValue(def)
}

func NewCertificateSyncChefResource() resource.Resource {
	return &CertificateSyncBaseResource{
		App:              infisical.CertificateSyncAppChef,
		SyncName:         "Chef",
		ResourceTypeName: "_certificate_sync_chef",
		AppConnection:    infisical.AppConnectionAppChef,
		DestinationConfigAttributes: map[string]schema.Attribute{
			"data_bag_name": schema.StringAttribute{
				Required:    true,
				CustomType:  customtypes.TrimmedStringType{},
				Description: "The name of the Chef data bag to sync certificates to. May only contain alphanumeric characters, underscores, and hyphens.",
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 255)},
			},
		},
		SyncOptionsAttributes: map[string]schema.Attribute{
			"certificate_name_schema": schema.StringAttribute{
				Required:    true,
				CustomType:  customtypes.TrimmedStringType{},
				Description: "The naming scheme for the synced data bag items. Must include the {{certificateId}} or {{shortCertificateId}} placeholder and compile to 1-255 alphanumeric characters, underscores, or hyphens. Available placeholders: {{certificateId}}, {{shortCertificateId}}, {{profileId}}, {{applicationId}}, {{applicationName}}, {{commonName}}.",
			},
			"can_remove_certificates": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether Infisical should remove data bag items from Chef when their certificates are no longer managed in Infisical. Defaults to `true`.",
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
				Description: "Whether a renewed certificate overwrites its existing data bag item instead of creating a new one. Defaults to `true`.",
				Default:     booldefault.StaticBool(true),
			},
			"update_existing_certificates": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether Infisical should update data bag items that already exist in Chef. Defaults to `true`.",
				Default:     booldefault.StaticBool(true),
			},
			"field_mappings": schema.SingleNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The data bag item field names that hold each part of the certificate. Defaults to `certificate`, `private_key`, `certificate_chain`, and `ca_certificate`.",
				Default: objectdefault.StaticValue(types.ObjectValueMust(certificateSyncChefFieldMappingsAttrTypes, map[string]attr.Value{
					"certificate":       types.StringValue(chefDefaultFieldCertificate),
					"private_key":       types.StringValue(chefDefaultFieldPrivateKey),
					"certificate_chain": types.StringValue(chefDefaultFieldCertificateChain),
					"ca_certificate":    types.StringValue(chefDefaultFieldCaCertificate),
				})),
				Attributes: map[string]schema.Attribute{
					"certificate":       chefFieldMappingAttribute("The field name that holds the certificate. Defaults to `certificate`.", chefDefaultFieldCertificate),
					"private_key":       chefFieldMappingAttribute("The field name that holds the private key. Defaults to `private_key`.", chefDefaultFieldPrivateKey),
					"certificate_chain": chefFieldMappingAttribute("The field name that holds the certificate chain. Defaults to `certificate_chain`.", chefDefaultFieldCertificateChain),
					"ca_certificate":    chefFieldMappingAttribute("The field name that holds the CA certificate. Defaults to `ca_certificate`.", chefDefaultFieldCaCertificate),
				},
			},
		},

		ReadSyncOptionsFromPlan: func(ctx context.Context, plan CertificateSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			var syncOptions CertificateSyncChefSyncOptionsModel
			diags := plan.SyncOptions.As(ctx, &syncOptions, objectAsOptions)
			if diags.HasError() {
				return nil, diags
			}

			var fieldMappings CertificateSyncChefFieldMappingsModel
			diags.Append(syncOptions.FieldMappings.As(ctx, &fieldMappings, objectAsOptions)...)
			if diags.HasError() {
				return nil, diags
			}

			return map[string]interface{}{
				"certificateNameSchema":      syncOptions.CertificateNameSchema.ValueString(),
				"canRemoveCertificates":      syncOptions.CanRemoveCertificates.ValueBool(),
				"includeRootCa":              syncOptions.IncludeRootCa.ValueBool(),
				"preserveItemOnRenewal":      syncOptions.PreserveItemOnRenewal.ValueBool(),
				"updateExistingCertificates": syncOptions.UpdateExistingCertificates.ValueBool(),
				"canImportCertificates":      false,
				"fieldMappings": map[string]interface{}{
					"certificate":      fieldMappings.Certificate.ValueString(),
					"privateKey":       fieldMappings.PrivateKey.ValueString(),
					"certificateChain": fieldMappings.CertificateChain.ValueString(),
					"caCertificate":    fieldMappings.CaCertificate.ValueString(),
				},
			}, diags
		},

		ReadSyncOptionsFromApi: func(_ context.Context, certificateSync infisical.CertificateSync) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			certificateNameSchema := trimmedStringFromMap(certificateSync.SyncOptions, "certificateNameSchema", &diags)
			if diags.HasError() {
				return types.ObjectNull(certificateSyncChefSyncOptionsAttrTypes), diags
			}

			apiFieldMappings, _ := certificateSync.SyncOptions["fieldMappings"].(map[string]interface{})
			fieldMappings, d := types.ObjectValue(certificateSyncChefFieldMappingsAttrTypes, map[string]attr.Value{
				"certificate":       chefFieldMappingFromMap(apiFieldMappings, "certificate", chefDefaultFieldCertificate),
				"private_key":       chefFieldMappingFromMap(apiFieldMappings, "privateKey", chefDefaultFieldPrivateKey),
				"certificate_chain": chefFieldMappingFromMap(apiFieldMappings, "certificateChain", chefDefaultFieldCertificateChain),
				"ca_certificate":    chefFieldMappingFromMap(apiFieldMappings, "caCertificate", chefDefaultFieldCaCertificate),
			})
			diags.Append(d...)
			if diags.HasError() {
				return types.ObjectNull(certificateSyncChefSyncOptionsAttrTypes), diags
			}

			return types.ObjectValue(certificateSyncChefSyncOptionsAttrTypes, map[string]attr.Value{
				"certificate_name_schema":      certificateNameSchema,
				"can_remove_certificates":      boolFromMap(certificateSync.SyncOptions, "canRemoveCertificates", true),
				"include_root_ca":              boolFromMap(certificateSync.SyncOptions, "includeRootCa", false),
				"preserve_item_on_renewal":     boolFromMap(certificateSync.SyncOptions, "preserveItemOnRenewal", true),
				"update_existing_certificates": boolFromMap(certificateSync.SyncOptions, "updateExistingCertificates", true),
				"field_mappings":               fieldMappings,
			})
		},

		ReadDestinationConfigFromPlan: func(ctx context.Context, plan CertificateSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			var destinationConfig CertificateSyncChefDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &destinationConfig, objectAsOptions)
			if diags.HasError() {
				return nil, diags
			}

			return map[string]interface{}{
				"dataBagName": destinationConfig.DataBagName.ValueString(),
			}, diags
		},

		ReadDestinationConfigFromApi: func(_ context.Context, certificateSync infisical.CertificateSync) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			dataBagName := trimmedStringFromMap(certificateSync.DestinationConfig, "dataBagName", &diags)
			if diags.HasError() {
				return types.ObjectNull(certificateSyncChefDestinationConfigAttrTypes), diags
			}

			return types.ObjectValue(certificateSyncChefDestinationConfigAttrTypes, map[string]attr.Value{
				"data_bag_name": dataBagName,
			})
		},
	}
}
