package resource

import (
	"context"
	"regexp"
	infisical "terraform-provider-infisical/internal/client"
	customtypes "terraform-provider-infisical/internal/pkg/customtypes"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type CertificateSyncAzureKeyVaultDestinationConfigModel struct {
	VaultBaseUrl types.String `tfsdk:"vault_base_url"`
}

type CertificateSyncAzureKeyVaultSyncOptionsModel struct {
	CertificateNameSchema customtypes.TrimmedStringValue `tfsdk:"certificate_name_schema"`
	CanRemoveCertificates types.Bool                     `tfsdk:"can_remove_certificates"`
	IncludeRootCa         types.Bool                     `tfsdk:"include_root_ca"`
	EnableVersioning      types.Bool                     `tfsdk:"enable_versioning"`
}

var certificateSyncAzureKeyVaultDestinationConfigAttrTypes = map[string]attr.Type{
	"vault_base_url": types.StringType,
}

var certificateSyncAzureKeyVaultSyncOptionsAttrTypes = map[string]attr.Type{
	"certificate_name_schema": customtypes.TrimmedStringType{},
	"can_remove_certificates": types.BoolType,
	"include_root_ca":         types.BoolType,
	"enable_versioning":       types.BoolType,
}

func NewCertificateSyncAzureKeyVaultResource() resource.Resource {
	return &CertificateSyncBaseResource{
		App:                 infisical.CertificateSyncAppAzureKeyVault,
		SyncName:            "Azure Key Vault",
		ResourceTypeName:    "_certificate_sync_azure_key_vault",
		CertificateNameRule: &certificateNameRule{pattern: regexp.MustCompile(`^[a-zA-Z0-9-]{1,127}$`), requireIdentifier: true, requirement: "1-127 letters, digits or hyphens"},
		AppConnection:       infisical.AppConnectionAppAzureKeyVault,
		DestinationConfigAttributes: map[string]schema.Attribute{
			"vault_base_url": schema.StringAttribute{
				Required:    true,
				Description: "The URL of the Azure Key Vault to sync certificates to (e.g. https://<vault-name>.vault.azure.net).",
			},
		},
		SyncOptionsAttributes: map[string]schema.Attribute{
			"certificate_name_schema": schema.StringAttribute{
				Required:    true,
				CustomType:  customtypes.TrimmedStringType{},
				Description: "The naming scheme for synced certificates. Must include the {{certificateId}} or {{shortCertificateId}} placeholder, and compiled names may contain only alphanumeric characters and hyphens (1-127 characters). Available placeholders: {{certificateId}}, {{shortCertificateId}}, {{profileId}}, {{applicationId}}, {{applicationName}}, {{commonName}}.",
			},
			"can_remove_certificates": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether Infisical should remove certificates from Azure Key Vault when they are no longer managed in Infisical. Defaults to true.",
				Default:     booldefault.StaticBool(true),
			},
			"include_root_ca": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to include the root CA certificate in the synced certificate chain. Defaults to false.",
				Default:     booldefault.StaticBool(false),
			},
			"enable_versioning": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether a renewed certificate is added as a new version of the existing Azure Key Vault certificate instead of a new certificate. Defaults to true.",
				Default:     booldefault.StaticBool(true),
			},
		},

		ReadSyncOptionsFromPlan: func(ctx context.Context, plan CertificateSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			var syncOptions CertificateSyncAzureKeyVaultSyncOptionsModel
			diags := plan.SyncOptions.As(ctx, &syncOptions, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			return map[string]interface{}{
				"certificateNameSchema": syncOptions.CertificateNameSchema.ValueString(),
				"canRemoveCertificates": syncOptions.CanRemoveCertificates.ValueBool(),
				"includeRootCa":         syncOptions.IncludeRootCa.ValueBool(),
				"enableVersioning":      syncOptions.EnableVersioning.ValueBool(),
				"canImportCertificates": false,
			}, diags
		},

		ReadSyncOptionsFromApi: func(_ context.Context, certificateSync infisical.CertificateSync) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			certificateNameSchema := trimmedStringFromMap(certificateSync.SyncOptions, "certificateNameSchema", &diags)
			if diags.HasError() {
				return types.ObjectNull(certificateSyncAzureKeyVaultSyncOptionsAttrTypes), diags
			}

			return types.ObjectValue(certificateSyncAzureKeyVaultSyncOptionsAttrTypes, map[string]attr.Value{
				"certificate_name_schema": certificateNameSchema,
				"can_remove_certificates": boolFromMap(certificateSync.SyncOptions, "canRemoveCertificates", true),
				"include_root_ca":         boolFromMap(certificateSync.SyncOptions, "includeRootCa", false),
				"enable_versioning":       boolFromMap(certificateSync.SyncOptions, "enableVersioning", true),
			})
		},

		ReadDestinationConfigFromPlan: func(ctx context.Context, plan CertificateSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			var destinationConfig CertificateSyncAzureKeyVaultDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &destinationConfig, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			return map[string]interface{}{
				"vaultBaseUrl": destinationConfig.VaultBaseUrl.ValueString(),
			}, diags
		},

		ReadDestinationConfigFromApi: func(_ context.Context, certificateSync infisical.CertificateSync) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			vaultBaseUrl := stringFromMap(certificateSync.DestinationConfig, "vaultBaseUrl", &diags)
			if diags.HasError() {
				return types.ObjectNull(certificateSyncAzureKeyVaultDestinationConfigAttrTypes), diags
			}

			return types.ObjectValue(certificateSyncAzureKeyVaultDestinationConfigAttrTypes, map[string]attr.Value{
				"vault_base_url": vaultBaseUrl,
			})
		},
	}
}
