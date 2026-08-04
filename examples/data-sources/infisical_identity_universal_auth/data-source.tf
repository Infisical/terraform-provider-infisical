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

data "infisical_identity_universal_auth" "example" {
  identity_id = "<identity-id>"
}

output "universal-auth-id" {
  value = data.infisical_identity_universal_auth.example.id
}

output "universal-auth-client-id" {
  value = data.infisical_identity_universal_auth.example.client_id
}
