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

resource "infisical_project_template" "example-project" {
  name        = "example-project"
  description = "This is an example project"
  type        = "secret-manager"
  environments = [{
    "name" : "development",
    "slug" : "dev",
    "position" : 1
  }]
}
