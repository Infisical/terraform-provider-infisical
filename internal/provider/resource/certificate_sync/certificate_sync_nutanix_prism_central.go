package resource

import (
	"context"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type CertificateSyncNutanixPrismCentralDestinationConfigModel struct {
	ClusterID   types.String `tfsdk:"cluster_id"`
	ClusterName types.String `tfsdk:"cluster_name"`
}

var certificateSyncNutanixPrismCentralDestinationConfigAttrTypes = map[string]attr.Type{
	"cluster_id":   types.StringType,
	"cluster_name": types.StringType,
}

func NewCertificateSyncNutanixPrismCentralResource() resource.Resource {
	return &CertificateSyncBaseResource{
		App:              infisical.CertificateSyncAppNutanixPrismCentral,
		SyncName:         "Nutanix Prism Central",
		ResourceTypeName: "_certificate_sync_nutanix_prism_central",
		AppConnection:    infisical.AppConnectionAppNutanixPrismCentral,
		DestinationConfigAttributes: map[string]schema.Attribute{
			"cluster_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the Nutanix cluster to install the certificate on.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Nutanix cluster to install the certificate on.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
		},

		MaxCertificates: 1,

		ReadDestinationConfigFromPlan: func(ctx context.Context, plan CertificateSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			var destinationConfig CertificateSyncNutanixPrismCentralDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &destinationConfig, objectAsOptions)
			if diags.HasError() {
				return nil, diags
			}

			return map[string]interface{}{
				"clusterId":   destinationConfig.ClusterID.ValueString(),
				"clusterName": destinationConfig.ClusterName.ValueString(),
			}, diags
		},

		ReadDestinationConfigFromApi: func(_ context.Context, certificateSync infisical.CertificateSync) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			clusterID := stringFromMap(certificateSync.DestinationConfig, "clusterId", &diags)
			clusterName := stringFromMap(certificateSync.DestinationConfig, "clusterName", &diags)
			if diags.HasError() {
				return types.ObjectNull(certificateSyncNutanixPrismCentralDestinationConfigAttrTypes), diags
			}

			return types.ObjectValue(certificateSyncNutanixPrismCentralDestinationConfigAttrTypes, map[string]attr.Value{
				"cluster_id":   clusterID,
				"cluster_name": clusterName,
			})
		},
	}
}
