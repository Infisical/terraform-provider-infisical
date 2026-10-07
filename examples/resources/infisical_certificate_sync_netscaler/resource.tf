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

resource "infisical_certificate_sync_netscaler" "example" {
  name           = "netscaler-certificate-sync-demo"
  description    = "Demo of NetScaler certificate sync"
  application_id = "<cert-manager-application-id>"
  connection_id  = "<app-connection-id>"

  certificate_filters = {
    certificate_ids = [infisical_cert_manager_certificate.example.id]
  }

  sync_options = {
    certificate_name_schema  = "Infisical-{{shortCertificateId}}" # Compiled names are limited to 63 characters
    can_remove_certificates  = true
    include_root_ca          = false
    preserve_item_on_renewal = true
  }

  destination_config = {
    vserver_name = "<ssl-vserver-name>" # Optional
  }
}
