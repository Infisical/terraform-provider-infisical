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

# Deprecated: list certificates in certificate_filters on the sync, then drop this with a removed block.
resource "infisical_certificate_sync_certificate" "example" {
  certificate_sync_id = infisical_certificate_sync_aws_certificate_manager.example.id
  certificate_id      = infisical_cert_manager_certificate.example.id
}
