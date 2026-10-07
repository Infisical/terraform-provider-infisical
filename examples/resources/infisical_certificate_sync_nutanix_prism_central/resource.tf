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

resource "infisical_certificate_sync_nutanix_prism_central" "example" {
  name           = "nutanix-prism-central-certificate-sync-demo"
  description    = "Demo of Nutanix Prism Central certificate sync"
  application_id = "<cert-manager-application-id>"
  connection_id  = "<app-connection-id>"

  # A Nutanix Prism Central cluster holds a single certificate.
  certificate_filters = {
    certificate_ids = [infisical_cert_manager_certificate.example.id]
  }

  destination_config = {
    cluster_id   = "<cluster-id>"
    cluster_name = "<cluster-name>"
  }
}
