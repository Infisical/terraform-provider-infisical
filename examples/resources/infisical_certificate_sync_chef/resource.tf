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

resource "infisical_certificate_sync_chef" "example" {
  name           = "chef-certificate-sync-demo"
  description    = "Demo of Chef certificate sync"
  application_id = "<cert-manager-application-id>"
  connection_id  = "<app-connection-id>"

  certificate_filters = {
    certificate_ids = [infisical_cert_manager_certificate.example.id]
  }

  sync_options = {
    certificate_name_schema      = "Infisical-{{certificateId}}" # Must include {{certificateId}} or {{shortCertificateId}}
    can_remove_certificates      = true
    include_root_ca              = false
    preserve_item_on_renewal     = true
    update_existing_certificates = true

    field_mappings = {
      certificate       = "certificate"
      private_key       = "private_key"
      certificate_chain = "certificate_chain"
      ca_certificate    = "ca_certificate"
    }
  }

  destination_config = {
    data_bag_name = "ssl_certificates"
  }
}
