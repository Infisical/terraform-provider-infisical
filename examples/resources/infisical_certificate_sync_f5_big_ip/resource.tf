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

resource "infisical_certificate_sync_f5_big_ip" "example" {
  name           = "f5-big-ip-certificate-sync-demo"
  description    = "Demo of F5 BIG-IP certificate sync"
  application_id = "<cert-manager-application-id>"
  connection_id  = "<app-connection-id>"

  certificate_filters = {
    certificate_ids = [infisical_cert_manager_certificate.example.id]
  }

  sync_options = {
    certificate_name_schema  = "Infisical-{{certificateId}}"
    can_remove_certificates  = true
    include_root_ca          = false
    preserve_item_on_renewal = true
  }

  destination_config = {
    partition                 = "Common"
    profile_type              = "client-ssl" # none, client-ssl, or server-ssl
    profile_name              = "infisical-clientssl"
    create_profile_if_missing = true
    parent_profile            = "/Common/clientssl" # Only allowed when create_profile_if_missing is true
  }
}
