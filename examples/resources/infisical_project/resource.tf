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
  name        = "GCP Project"
  slug        = "gcp-project"
  description = "This is a GCP project"
  type        = "secret-manager" # Default project type
}

resource "infisical_project" "aws-project" {
  name        = "AWS Project"
  slug        = "aws-project"
  description = "This is an AWS project"
  type        = "secret-manager"
}

resource "infisical_project" "kms-project" {
  name        = "KMS Project"
  slug        = "kms-project"
  description = "This is a KMS project for key management"
  type        = "kms"
}

resource "infisical_project" "certificate-project" {
  name        = "Certificates Project"
  slug        = "certificate-project"
  description = "This is a certificates project for certificate management"
  type        = "cert-manager"
}

# Keep the description out of the plan output and state (requires Terraform 1.11.0 or higher).
# Increment description_wo_version to apply a new description_wo.
resource "infisical_project" "private-project" {
  name                   = "Private Project"
  slug                   = "private-project"
  description_wo         = "Owner: platform-team@example.com"
  description_wo_version = 1
}
