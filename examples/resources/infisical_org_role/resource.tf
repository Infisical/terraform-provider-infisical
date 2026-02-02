terraform {
  required_providers {
    infisical = {
      # version = <latest version>
      source = "infisical/infisical"
    }
  }
}

provider "infisical" {
  host = "http://app.infisical.com" # Only required if using self hosted instance of Infisical, default is https://app.infisical.com
  auth = {
    universal = {
      client_id     = "<machine-identity-client-id>"
      client_secret = "<machine-identity-client-secret>"
    }
  }
}

resource "infisical_org_role" "tester" {
  name        = "Tester"
  description = "A test role"
  slug        = "tester"
  permissions = [
    {
      subject = "project"
      action  = ["create"]
    },
    {
      subject = "app-connections"
      action  = ["read", "create"]
      conditions = jsonencode({
        connectionId = {
          "$eq" = "<connection-id>"
        }
      })
    },
  ]
}

