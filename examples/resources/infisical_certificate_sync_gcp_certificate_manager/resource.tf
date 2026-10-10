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

resource "infisical_certificate_sync_gcp_certificate_manager" "example" {
  name           = "gcp-certificate-manager-certificate-sync-demo"
  description    = "Demo of GCP Certificate Manager certificate sync"
  application_id = "<cert-manager-application-id>"
  connection_id  = "<app-connection-id>"

  certificate_filters = {
    certificate_ids = [infisical_cert_manager_certificate.example.id]
  }

  sync_options = {
    certificate_name_schema  = "infisical-{{certificateId}}" # Must start with a lowercase letter
    can_remove_certificates  = true
    include_root_ca          = false
    preserve_item_on_renewal = true
    labels = {
      environment = "production"
    }
  }

  destination_config = {
    gcp_project_id = "my-gcp-project"
    location       = "global"
    scope          = "default"
    certificate_map_binding = {
      certificate_map = "my-certificate-map"
      hostname        = "www.example.com"
    }
  }
}
