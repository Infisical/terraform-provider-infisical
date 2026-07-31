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

resource "infisical_identity" "machine-identity-1" {
  name   = "machine-identity-1"
  role   = "admin"
  org_id = "<your-org-id>"
}

resource "infisical_identity_tls_cert_auth" "tls-cert-auth" {
  identity_id    = infisical_identity.machine-identity-1.id
  ca_certificate = file("${path.module}/ca.pem")

  allowed_common_names      = ["my-service.example.com"]
  allowed_subject_alt_names = ["URI:spiffe://example.org/my-service"]

  verify_client_certificate_chain = false

  access_token_ttl     = 2592000
  access_token_max_ttl = 2592000

  access_token_trusted_ips = [
    { ip_address = "0.0.0.0/0" }
  ]
}
