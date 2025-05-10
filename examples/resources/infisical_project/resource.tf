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

resource "infisical_project" "gcp-project" {
  name                  = "GCP Project"
  slug                  = "gcp-project"
  description           = "This is a GCP project"
  has_delete_protection = true
}

resource "infisical_project" "aws-project" {
  name                  = "AWS Project"
  slug                  = "aws-project"
  description           = "This is an AWS project"
  has_delete_protection = false
}

resource "infisical_project" "azure-project" {
  name = "Azure Project"
  slug = "azure-project"
}
