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
  name     = "example"
  slug     = "example"
  position = 1 # Optional
}

resource "infisical_project_environment" "pre-prod" {
  name       = "pre-prod"
  project_id = infisical_project.example.id
  slug       = "preprod"
  position   = 2 # Optional
}
