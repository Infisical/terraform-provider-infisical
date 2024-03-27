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
  client_id     = "<machine-identity-client-id>"
  client_secret = "<machine-identity-client-secret>"
}

resource "infisical_project" "gcp-project" {
  name = "GCP Project"
  slug = "gcp-project"
}

resource "infisical_project" "aws-project" {
  name = "AWS Project"
  slug = "aws-project"
}

resource "infisical_project" "azure-project" {
  name = "Azure Project"
  slug = "azure-project"
}


