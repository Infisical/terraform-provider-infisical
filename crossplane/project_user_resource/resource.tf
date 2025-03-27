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

resource "infisical_project_user" "test-user" {
  project_id = infisical_project.example.id
  username   = "sheen+200@infisical.com"
  roles = jsonencode([
    {
      role_slug = "admin"
    },
    {
      role_slug = "no-access"
    }
  ])
}
