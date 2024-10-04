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


resource "infisical_integration_circleci" "circleci-integration" {
  project_id  = "225393b9-e3d6-424f-9df3-22c3cdeb97c9"
  environment = "dev"
  secret_path = "/test-folder"

  circleci_token      = "<your-circle-cipersonal-access-token>"
  circleci_project_id = "<your-circleci-project-id>"
  circleci_org_slug   = "<your-circleci-org-slug>"
}