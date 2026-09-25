package resource

import (
	infisical "terraform-provider-infisical/internal/client"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func NewAuditLogStreamSumoLogicResource() resource.Resource {
	return &AuditLogStreamBaseResource{
		Provider:         infisical.AuditLogStreamProviderSumoLogic,
		ProviderName:     "Sumo Logic",
		ResourceTypeName: "_audit_log_stream_sumo_logic",
		CredentialFields: []credentialField{
			{
				Name:        "url",
				JSONName:    "url",
				Description: "The HTTP Logs and Metrics Source base URL (e.g. https://endpoint4.collection.sumologic.com/receiver/v1/http).",
				Validators:  []validator.String{infisicaltf.HttpsUrlValidator},
			},
			{
				Name:          "token",
				JSONName:      "token",
				Sensitive:     true,
				MaskUnchanged: true,
				Description:   "The source authentication token, taken from the x-sumo-token header Sumo Logic displays for the source. Applies that do not change it leave the stored token alone, so a rotation done in Infisical survives.",
			},
		},
	}
}
