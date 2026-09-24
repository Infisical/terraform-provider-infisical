package resource

import (
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func NewAuditLogStreamCustomResource() resource.Resource {
	return &AuditLogStreamBaseResource{
		Provider:         infisical.AuditLogStreamProviderCustom,
		ProviderName:     "a custom HTTP endpoint",
		ResourceTypeName: "_audit_log_stream_custom",
		CredentialFields: []credentialField{
			{
				Name:        "url",
				JSONName:    "url",
				Description: "The endpoint that receives batched audit logs as a JSON array. HTTPS is strongly recommended.",
			},
			{
				Name:        "headers",
				JSONName:    "headers",
				Kind:        credentialHeaderMap,
				Sensitive:   true,
				Description: "Request headers to send, keyed by header name. Usually carries the destination's auth token.",
			},
		},
	}
}
