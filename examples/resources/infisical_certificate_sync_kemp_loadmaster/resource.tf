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

resource "infisical_certificate_sync_kemp_loadmaster" "example" {
  name           = "kemp-loadmaster-certificate-sync-demo"
  description    = "Demo of Kemp LoadMaster certificate sync"
  application_id = "<cert-manager-application-id>"
  connection_id  = "<app-connection-id>"

  certificate_filters = {
    certificate_ids = [infisical_cert_manager_certificate.example.id]
  }

  sync_options = {
    certificate_name_schema    = "Infisical-{{certificateId}}" # Must include {{certificateId}} or {{shortCertificateId}}
    ca_certificate_name_schema = "Infisical-ca-{{fingerprint}}"
    can_remove_certificates    = true
    include_root_ca            = false
    preserve_item_on_renewal   = true
  }

  destination_config = {
    virtual_service_id = "1" # Optional numeric Virtual Service index
  }
}
