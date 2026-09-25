package resource

import (
	infisical "terraform-provider-infisical/internal/client"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func NewAuditLogStreamCustomResource() resource.Resource {
	return &AuditLogStreamBaseResource{
		Provider:         infisical.AuditLogStreamProviderCustom,
		ProviderName:     "a custom HTTP endpoint",
		ResourceTypeName: "_audit_log_stream_custom",
		CredentialFields: []credentialField{
			{
				Name:     "url",
				JSONName: "url",
				Description: "The endpoint that receives batched audit logs as a JSON array. http is accepted for collectors on a " +
					"trusted private network, but sends audit events and the headers below in the clear and warns at plan time; prefer https.",
				Validators: []validator.String{infisicaltf.HttpsPreferredUrlValidator},
			},
			{
				Name:          "headers",
				JSONName:      "headers",
				Kind:          credentialHeaderMap,
				Sensitive:     true,
				MaskUnchanged: true,
				Description:   "Request headers to send, keyed by header name. Usually carries the destination's auth token. Applies leave unchanged header values alone, so a rotation done in Infisical survives.",
			},
		},
	}
}
