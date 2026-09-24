package resource

import (
	infisical "terraform-provider-infisical/internal/client"
	infisicaltf "terraform-provider-infisical/internal/pkg/terraform"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func NewAuditLogStreamAzureResource() resource.Resource {
	return &AuditLogStreamBaseResource{
		Provider:         infisical.AuditLogStreamProviderAzure,
		ProviderName:     "Azure Monitor",
		ResourceTypeName: "_audit_log_stream_azure",
		CredentialFields: []credentialField{
			{
				Name:        "tenant_id",
				JSONName:    "tenantId",
				Description: "The Entra ID tenant ID.",
			},
			{
				Name:        "client_id",
				JSONName:    "clientId",
				Description: "The Entra ID application (client) ID.",
			},
			{
				Name:        "client_secret",
				JSONName:    "clientSecret",
				Sensitive:   true,
				Description: "The Entra ID application client secret. Must be 40 characters.",
			},
			{
				Name:        "dce_url",
				JSONName:    "dceUrl",
				Description: "The Data Collection Endpoint URL.",
				Validators:  []validator.String{infisicaltf.HttpsUrlValidator},
			},
			{
				Name:        "dcr_id",
				JSONName:    "dcrId",
				Description: "The Data Collection Rule immutable ID, in dcr-<32 hex characters> format.",
			},
			{
				Name:        "clt_name",
				JSONName:    "cltName",
				Description: "The custom log table name to write to.",
			},
		},
	}
}
