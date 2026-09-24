package resource

import (
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func NewAuditLogStreamSplunkResource() resource.Resource {
	return &AuditLogStreamBaseResource{
		Provider:         infisical.AuditLogStreamProviderSplunk,
		ProviderName:     "Splunk",
		ResourceTypeName: "_audit_log_stream_splunk",
		CredentialFields: []credentialField{
			{
				Name:        "hostname",
				JSONName:    "hostname",
				Description: "The HTTP Event Collector hostname, without protocol, port or path (e.g. http-inputs-acme.splunkcloud.com).",
			},
			{
				Name:        "port",
				JSONName:    "port",
				Kind:        credentialInt,
				Optional:    true,
				Description: "The HTTP Event Collector port. Defaults to 8088 (Splunk Enterprise); Splunk Cloud serves the collector on 443.",
			},
			{
				Name:        "token",
				JSONName:    "token",
				Sensitive:   true,
				Description: "The HTTP Event Collector token. Must be a UUID.",
			},
		},
	}
}
