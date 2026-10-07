package resource

import (
	"context"
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

type CertificateSyncCloudflareCustomCertificateDestinationConfigModel struct {
	ZoneID types.String `tfsdk:"zone_id"`
}

type CertificateSyncCloudflareCustomCertificateSyncOptionsModel struct {
	CertificateNameSchema customtypes.TrimmedStringValue `tfsdk:"certificate_name_schema"`
	CanRemoveCertificates types.Bool                     `tfsdk:"can_remove_certificates"`
}

var certificateSyncCloudflareCustomCertificateDestinationConfigAttrTypes = map[string]attr.Type{
	"zone_id": types.StringType,
}

var certificateSyncCloudflareCustomCertificateSyncOptionsAttrTypes = map[string]attr.Type{
	"certificate_name_schema": customtypes.TrimmedStringType{},
	"can_remove_certificates": types.BoolType,
}

func NewCertificateSyncCloudflareCustomCertificateResource() resource.Resource {
	return &CertificateSyncBaseResource{
		App:              infisical.CertificateSyncAppCloudflareCustomCertificate,
		SyncName:         "Cloudflare Custom SSL Certificate",
		ResourceTypeName: "_certificate_sync_cloudflare_custom_certificate",
		AppConnection:    infisical.AppConnectionAppCloudflare,
		DestinationConfigAttributes: map[string]schema.Attribute{
			"zone_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the Cloudflare zone to upload custom SSL certificates to.",
			},
		},
		SyncOptionsAttributes: map[string]schema.Attribute{
			"certificate_name_schema": schema.StringAttribute{
				Required:    true,
				CustomType:  customtypes.TrimmedStringType{},
				Description: "The naming scheme for synced certificates. Must include the {{certificateId}} or {{shortCertificateId}} placeholder, and compiled names may contain only alphanumeric characters, hyphens and underscores (1-255 characters). Available placeholders: {{certificateId}}, {{shortCertificateId}}, {{profileId}}, {{applicationId}}, {{applicationName}}, {{commonName}}.",
			},
			"can_remove_certificates": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether Infisical should remove custom certificates from Cloudflare when they are no longer managed in Infisical. Defaults to true.",
				Default:     booldefault.StaticBool(true),
			},
		},

		ReadSyncOptionsFromPlan: func(ctx context.Context, plan CertificateSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			var syncOptions CertificateSyncCloudflareCustomCertificateSyncOptionsModel
			diags := plan.SyncOptions.As(ctx, &syncOptions, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			return map[string]interface{}{
				"certificateNameSchema": syncOptions.CertificateNameSchema.ValueString(),
				"canRemoveCertificates": syncOptions.CanRemoveCertificates.ValueBool(),
			}, diags
		},

		ReadSyncOptionsFromApi: func(_ context.Context, certificateSync infisical.CertificateSync) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			certificateNameSchema := trimmedStringFromMap(certificateSync.SyncOptions, "certificateNameSchema", &diags)
			if diags.HasError() {
				return types.ObjectNull(certificateSyncCloudflareCustomCertificateSyncOptionsAttrTypes), diags
			}

			return types.ObjectValue(certificateSyncCloudflareCustomCertificateSyncOptionsAttrTypes, map[string]attr.Value{
				"certificate_name_schema": certificateNameSchema,
				"can_remove_certificates": boolFromMap(certificateSync.SyncOptions, "canRemoveCertificates", true),
			})
		},

		ReadDestinationConfigFromPlan: func(ctx context.Context, plan CertificateSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			var destinationConfig CertificateSyncCloudflareCustomCertificateDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &destinationConfig, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			return map[string]interface{}{
				"zoneId": destinationConfig.ZoneID.ValueString(),
			}, diags
		},

		ReadDestinationConfigFromApi: func(_ context.Context, certificateSync infisical.CertificateSync) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			zoneID := stringFromMap(certificateSync.DestinationConfig, "zoneId", &diags)
			if diags.HasError() {
				return types.ObjectNull(certificateSyncCloudflareCustomCertificateDestinationConfigAttrTypes), diags
			}

			return types.ObjectValue(certificateSyncCloudflareCustomCertificateDestinationConfigAttrTypes, map[string]attr.Value{
				"zone_id": zoneID,
			})
		},
	}
}
