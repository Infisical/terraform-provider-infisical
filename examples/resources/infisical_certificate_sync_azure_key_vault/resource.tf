terraform {
  required_providers {
    infisical = {
      # version = <latest version>
      source = "infisical/infisical"
    }
  }
}

provider "infisical" {
  host = "https://app.infisical.com" # Only required if using self hosted instance of Infisical, default is https://app.infisical.com
  auth = {
    universal = {
      client_id     = "<machine-identity-client-id>"
      client_secret = "<machine-identity-client-secret>"
    }
  }
}

resource "infisical_certificate_sync_azure_key_vault" "example" {
  name           = "azure-key-vault-certificate-sync-demo"
  description    = "Demo of Azure Key Vault certificate sync"
  application_id = "<cert-manager-application-id>"
  connection_id  = "<app-connection-id>"

  certificate_filters = {
    certificate_ids = [infisical_cert_manager_certificate.example.id]
  }

  sync_options = {
    certificate_name_schema = "infisical-{{certificateId}}" # Alphanumeric characters and hyphens only
    can_remove_certificates = true
    include_root_ca         = false
    enable_versioning       = true
  }

  destination_config = {
    vault_base_url = "https://my-vault.vault.azure.net"
  }
}
