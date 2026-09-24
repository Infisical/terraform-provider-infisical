package resource

import (
	infisical "terraform-provider-infisical/internal/client"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func NewAuditLogStreamCriblResource() resource.Resource {
	return &AuditLogStreamBaseResource{
		Provider:         infisical.AuditLogStreamProviderCribl,
		ProviderName:     "Cribl",
		ResourceTypeName: "_audit_log_stream_cribl",
		CredentialFields: []credentialField{
			{
				Name:        "url",
				JSONName:    "url",
				Description: "The Cribl Stream HTTP source URL.",
				Validators:  []validator.String{infisicaltf.HttpsUrlValidator},
			},
			{
				Name:        "token",
				JSONName:    "token",
				Sensitive:   true,
				Description: "The Cribl Stream HTTP source token.",
			},
		},
	}
}
