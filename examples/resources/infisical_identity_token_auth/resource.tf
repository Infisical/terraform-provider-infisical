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

resource "infisical_identity_token_auth" "token-auth" {
  identity_id     = infisical_identity.machine-identity-1.id
}

