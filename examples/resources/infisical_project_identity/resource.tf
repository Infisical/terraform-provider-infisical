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

resource "infisical_project_identity" "test-identity" {
  project_id = infisical_project.example.id
  identity_id = "<identity id>"
  roles = [
    {
      role_slug = "admin"
    }
  ]
}
