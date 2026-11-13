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

resource "infisical_certificate_sync_palo_alto_networks_ssl_tls_profile" "example" {
  name           = "palo-alto-networks-ssl-tls-profile-sync-demo"
  description    = "Demo of Palo Alto Networks SSL/TLS profile certificate sync"
  application_id = "<cert-manager-application-id>"
  connection_id  = "<app-connection-id>"

  certificate_filters = {
    # This sync holds a single certificate. Change the order ID to swap the profile's certificate.
    certificate_order_ids = [infisical_cert_manager_certificate.example.order_id]
  }

  sync_options = {
    certificate_name_schema  = "INF-{{shortCertificateId}}"
    can_remove_certificates  = true
    include_root_ca          = false
    preserve_item_on_renewal = true
  }

  destination_config = {
    ssl_tls_service_profile_name = "infisical-ssl-profile"
    ssl_tls_service_profile_vsys = "vsys1" # Leave unset when the profile is in the shared scope
    # template                   = "<panorama-template>" # Required when the connection is a Panorama, not allowed for a firewall
    push_to_devices = true
  }
}
