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

resource "infisical_project" "example" {
  name = "example"
  slug = "example"
}

resource "infisical_project_identity" "test-identity" {
  project_id  = infisical_project.example.id
  identity_id = "<identity id>"
  roles = [
    {
      role_slug = "admin"
    }
  ]
}

resource "infisical_project_identity_specific_privilege" "test-privilege" {
  project_slug = infisical_project.example.slug
  identity_id  = infisical_project_identity.test-identity.identity_id
  permissions_v2 = [
    {
      action   = ["edit"]
      subject  = "secret-folders",
      inverted = true,
    },
    {
      action  = ["read", "edit"]
      subject = "secrets",
      conditions = jsonencode({
        environment = {
          "$in" = ["dev", "prod"]
          "$eq" = "dev"
        }
        secretPath = {
          "$eq" = "/"
        }
      })
    },
  ]
}
