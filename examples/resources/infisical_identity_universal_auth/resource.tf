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

resource "infisical_identity" "example" {
  name   = "my-machine-identity"
  role   = "no-access"
  org_id = "<your-org-id>"
}

resource "infisical_identity_universal_auth" "example" {
  identity_id = infisical_identity.example.id

  access_token_ttl              = 2592000
  access_token_max_ttl          = 2592000
  access_token_num_uses_limit   = 0
  access_token_trusted_ips      = [{ ip_address = "0.0.0.0/0" }]
  client_secret_trusted_ips     = [{ ip_address = "0.0.0.0/0" }]
}
