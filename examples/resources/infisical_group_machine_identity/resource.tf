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
  name   = "example-identity"
  role   = "member"
  org_id = "<org-id>"
}

resource "infisical_group_machine_identity" "example" {
  group_id    = "<group-id>"
  identity_id = infisical_identity.example.id
}
