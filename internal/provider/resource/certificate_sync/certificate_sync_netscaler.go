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
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type CertificateSyncNetScalerDestinationConfigModel struct {
	VserverName types.String `tfsdk:"vserver_name"`
}

type CertificateSyncNetScalerSyncOptionsModel struct {
	CertificateNameSchema customtypes.TrimmedStringValue `tfsdk:"certificate_name_schema"`
	CanRemoveCertificates types.Bool                     `tfsdk:"can_remove_certificates"`
	IncludeRootCa         types.Bool                     `tfsdk:"include_root_ca"`
	PreserveItemOnRenewal types.Bool                     `tfsdk:"preserve_item_on_renewal"`
}

var certificateSyncNetScalerDestinationConfigAttrTypes = map[string]attr.Type{
	"vserver_name": types.StringType,
}

var certificateSyncNetScalerSyncOptionsAttrTypes = map[string]attr.Type{
	"certificate_name_schema":  customtypes.TrimmedStringType{},
	"can_remove_certificates":  types.BoolType,
	"include_root_ca":          types.BoolType,
	"preserve_item_on_renewal": types.BoolType,
}

func NewCertificateSyncNetScalerResource() resource.Resource {
	return &CertificateSyncBaseResource{
		App:              infisical.CertificateSyncAppNetScaler,
		SyncName:         "NetScaler",
		ResourceTypeName: "_certificate_sync_netscaler",
		AppConnection:    infisical.AppConnectionAppNetScaler,
		DestinationConfigAttributes: map[string]schema.Attribute{
			"vserver_name": schema.StringAttribute{
				Optional:    true,
				Description: "The name of the NetScaler SSL vServer to bind synced certificates to. Leave unset to sync certificates without binding them to a vServer.",
				Validators:  []validator.String{stringvalidator.LengthAtMost(127)},
			},
		},
		SyncOptionsAttributes: map[string]schema.Attribute{
			"certificate_name_schema": schema.StringAttribute{
				Required:    true,
				CustomType:  customtypes.TrimmedStringType{},
				Description: "The naming scheme for synced certkeys, which must compile to 1-63 alphanumeric characters, hyphens, underscores, or periods. Available placeholders: {{certificateId}}, {{shortCertificateId}}, {{profileId}}, {{applicationId}}, {{applicationName}}, {{commonName}}. A schema with no placeholder can be linked to only one certificate.",
			},
			"can_remove_certificates": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether Infisical should remove certificates from NetScaler when they are no longer managed in Infisical. Defaults to `true`.",
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
				Description: "Whether a renewed certificate replaces its existing NetScaler certkey instead of creating a new one. Defaults to `true`.",
				Default:     booldefault.StaticBool(true),
			},
		},

		ReadSyncOptionsFromPlan: func(ctx context.Context, plan CertificateSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			var syncOptions CertificateSyncNetScalerSyncOptionsModel
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
		},

		ReadSyncOptionsFromApi: func(_ context.Context, certificateSync infisical.CertificateSync) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			certificateNameSchema := trimmedStringFromMap(certificateSync.SyncOptions, "certificateNameSchema", &diags)
			if diags.HasError() {
				return types.ObjectNull(certificateSyncNetScalerSyncOptionsAttrTypes), diags
			}

			return types.ObjectValue(certificateSyncNetScalerSyncOptionsAttrTypes, map[string]attr.Value{
				"certificate_name_schema":  certificateNameSchema,
				"can_remove_certificates":  boolFromMap(certificateSync.SyncOptions, "canRemoveCertificates", true),
				"include_root_ca":          boolFromMap(certificateSync.SyncOptions, "includeRootCa", false),
				"preserve_item_on_renewal": boolFromMap(certificateSync.SyncOptions, "preserveItemOnRenewal", true),
			})
		},

		ReadDestinationConfigFromPlan: func(ctx context.Context, plan CertificateSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			var destinationConfig CertificateSyncNetScalerDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &destinationConfig, objectAsOptions)
			if diags.HasError() {
				return nil, diags
			}

			request := map[string]interface{}{}
			setOptionalString(request, "vserverName", destinationConfig.VserverName)
			return request, diags
		},

		ReadDestinationConfigFromApi: func(_ context.Context, certificateSync infisical.CertificateSync) (types.Object, diag.Diagnostics) {
			return types.ObjectValue(certificateSyncNetScalerDestinationConfigAttrTypes, map[string]attr.Value{
				"vserver_name": optionalStringFromMap(certificateSync.DestinationConfig, "vserverName"),
			})
		},
	}
}
