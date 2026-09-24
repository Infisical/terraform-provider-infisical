package resource

import (
	infisical "terraform-provider-infisical/internal/client"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func NewAuditLogStreamDatadogResource() resource.Resource {
	return &AuditLogStreamBaseResource{
		Provider:         infisical.AuditLogStreamProviderDatadog,
		ProviderName:     "Datadog",
		ResourceTypeName: "_audit_log_stream_datadog",
		CredentialFields: []credentialField{
			{
				Name:        "url",
				JSONName:    "url",
				Description: "The Datadog log intake URL for your site (e.g. https://http-intake.logs.datadoghq.com/api/v2/logs).",
				Validators:  []validator.String{infisicaltf.HttpsUrlValidator},
			},
			{
				Name:        "token",
				JSONName:    "token",
				Sensitive:   true,
				Description: "The Datadog API key. Must be 32 hexadecimal characters.",
			},
		},
	}
}
