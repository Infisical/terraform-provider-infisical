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

resource "infisical_certificate_sync_cloudflare_custom_certificate" "example" {
  name           = "cloudflare-certificate-sync-demo"
  description    = "Demo of Cloudflare Custom SSL Certificate sync"
  application_id = "<cert-manager-application-id>"
  connection_id  = "<app-connection-id>"

  certificate_filters = {
    certificate_ids = [infisical_cert_manager_certificate.example.id]
  }

  sync_options = {
    certificate_name_schema = "infisical-{{certificateId}}" # Must include {{certificateId}} or {{shortCertificateId}}
    can_remove_certificates = true
  }

  destination_config = {
    zone_id = "023e105f4ecef8ad9ca31a8372d0c353"
  }
}
