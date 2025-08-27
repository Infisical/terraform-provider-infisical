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

resource "infisical_app_connection_gitlab" "gitlab_connection" {
  name        = "gitlab-connection"
  description = "GitLab connection for CI/CD variables sync"
  method      = "access-token"

  credentials = {
    access_token      = "<access-token>"
    instance_url      = "https://gitlab.com" # Or your self-hosted GitLab URL
    access_token_type = "project" # Or "personal"
  }
}
