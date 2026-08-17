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

# Attach a certificate to a certificate sync so it is synced to the destination.
# The certificate must belong to the same application as the certificate sync.
#
# Only a currently valid certificate can be synced. Certificates that have been renewed,
# revoked, or expired are rejected by Infisical. When a certificate is renewed, Infisical
# attaches the renewed certificate to the sync and drops the superseded one, so a literal
# certificate ID will stop working at the next renewal.
#
# Reference the resource that issues or renews the certificate so the ID stays in step
# automatically, rather than hardcoding it:
resource "infisical_certificate_sync_certificate" "example" {
  certificate_sync_id = infisical_certificate_sync_aws_certificate_manager.example.id
  certificate_id      = infisical_cert_manager_certificate.example.id
}
