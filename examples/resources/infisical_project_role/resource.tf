terraform {
  required_providers {
    infisical = {
      # version = <latest version>
      source = "infisical/infisical"
    }
  }
}

provider "infisical" {
  host          = "https://app.infisical.com" # Only required if using self hosted instance of Infisical, default is https://app.infisical.com
  client_id     = "<>"
  client_secret = "<>"
}

resource "infisical_project" "example" {
  name = "example"
  slug = "example"
}

resource "infisical_project_role" "biller" {
  project_slug = infisical_project.example.slug
  name         = "Tester"
  description  = "A test role"
  slug         = "tester"
  permissions_v2 = [
    {
      subject = "integrations"
      action  = ["read", "create"]
    },
    {
      subject = "secrets"
      action  = ["read", "edit"]
      conditions = jsonencode({
        environment = {
          "$in" = ["dev", "prod"]
          "$eq" = "dev"
        }
      })
    },
  ]
}
