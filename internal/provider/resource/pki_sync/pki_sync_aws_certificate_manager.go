package resource

import (
	"context"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type PkiSyncAwsCertificateManagerDestinationConfigModel struct {
	Region types.String `tfsdk:"aws_region"`
}

type PkiSyncAwsCertificateManagerSyncOptionsModel struct {
	CertificateNameSchema types.String `tfsdk:"certificate_name_schema"`
	CanRemoveCertificates types.Bool   `tfsdk:"can_remove_certificates"`
	IncludeRootCa         types.Bool   `tfsdk:"include_root_ca"`
	PreserveArn           types.Bool   `tfsdk:"preserve_arn"`
}

var pkiSyncAwsCertificateManagerDestinationConfigAttrTypes = map[string]attr.Type{
	"aws_region": types.StringType,
}

var pkiSyncAwsCertificateManagerSyncOptionsAttrTypes = map[string]attr.Type{
	"certificate_name_schema": types.StringType,
	"can_remove_certificates": types.BoolType,
	"include_root_ca":         types.BoolType,
	"preserve_arn":            types.BoolType,
}

func NewPkiSyncAwsCertificateManagerResource() resource.Resource {
	return &PkiSyncBaseResource{
		App:              infisical.PkiSyncAppAWSCertificateManager,
		SyncName:         "AWS Certificate Manager",
		ResourceTypeName: "_pki_sync_aws_certificate_manager",
		AppConnection:    infisical.AppConnectionAppAWS,
		DestinationConfigAttributes: map[string]schema.Attribute{
			"aws_region": schema.StringAttribute{
				Required:    true,
				Description: "The AWS region to sync certificates to (e.g. us-east-1).",
			},
		},
		SyncOptionsAttributes: map[string]schema.Attribute{
			"certificate_name_schema": schema.StringAttribute{
				Required:    true,
				Description: "The naming scheme for synced certificates. Must include the {{certificateId}} or {{shortCertificateId}} placeholder. Available placeholders: {{certificateId}}, {{shortCertificateId}}, {{profileId}}, {{applicationId}}, {{applicationName}}, {{commonName}}.",
			},
			"can_remove_certificates": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether Infisical should remove certificates from AWS Certificate Manager when they are no longer managed in Infisical.",
				Default:     booldefault.StaticBool(true),
			},
			"include_root_ca": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to include the root CA certificate in the synced certificate chain.",
				Default:     booldefault.StaticBool(false),
			},
			"preserve_arn": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to preserve the AWS Certificate Manager ARN when a certificate is renewed, reimporting into the existing certificate instead of creating a new one.",
				Default:     booldefault.StaticBool(true),
			},
		},

		ReadSyncOptionsFromPlan: func(ctx context.Context, plan PkiSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			var syncOptions PkiSyncAwsCertificateManagerSyncOptionsModel
			diags := plan.SyncOptions.As(ctx, &syncOptions, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			return map[string]interface{}{
				"certificateNameSchema": syncOptions.CertificateNameSchema.ValueString(),
				"canRemoveCertificates": syncOptions.CanRemoveCertificates.ValueBool(),
				"includeRootCa":         syncOptions.IncludeRootCa.ValueBool(),
				"preserveArn":           syncOptions.PreserveArn.ValueBool(),
				// AWS Certificate Manager does not support importing certificates.
				"canImportCertificates": false,
			}, diags
		},

		ReadSyncOptionsFromApi: func(_ context.Context, pkiSync infisical.PkiSync) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			certificateNameSchema := stringFromMap(pkiSync.SyncOptions, "certificateNameSchema", &diags)
			if diags.HasError() {
				return types.ObjectNull(pkiSyncAwsCertificateManagerSyncOptionsAttrTypes), diags
			}

			return types.ObjectValue(pkiSyncAwsCertificateManagerSyncOptionsAttrTypes, map[string]attr.Value{
				"certificate_name_schema": certificateNameSchema,
				"can_remove_certificates": boolFromMap(pkiSync.SyncOptions, "canRemoveCertificates", true),
				"include_root_ca":         boolFromMap(pkiSync.SyncOptions, "includeRootCa", false),
				"preserve_arn":            boolFromMap(pkiSync.SyncOptions, "preserveArn", true),
			})
		},

		ReadDestinationConfigFromPlan: func(ctx context.Context, plan PkiSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			var destinationConfig PkiSyncAwsCertificateManagerDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &destinationConfig, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			return map[string]interface{}{
				"region": destinationConfig.Region.ValueString(),
			}, diags
		},

		ReadDestinationConfigFromApi: func(_ context.Context, pkiSync infisical.PkiSync) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			region := stringFromMap(pkiSync.DestinationConfig, "region", &diags)
			if diags.HasError() {
				return types.ObjectNull(pkiSyncAwsCertificateManagerDestinationConfigAttrTypes), diags
			}

			return types.ObjectValue(pkiSyncAwsCertificateManagerDestinationConfigAttrTypes, map[string]attr.Value{
				"aws_region": region,
			})
		},
	}
}
