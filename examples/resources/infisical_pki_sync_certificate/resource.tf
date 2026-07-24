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

# Associate a certificate with a PKI sync so it is synced to the destination.
# The certificate must belong to the same application as the PKI sync.
resource "infisical_pki_sync_certificate" "example" {
  pki_sync_id    = infisical_pki_sync_aws_certificate_manager.example.id
  certificate_id = "<certificate-id>"
}
