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

resource "infisical_app_connection_github" "github_connection" {
  name        = "github-connection"
  description = "GitHub connection for Actions secrets sync"
  method      = "pat"

  credentials = {
    personal_access_token = "<personal-access-token>"
    # instance_type = "cloud"  # Optional: "cloud" (default) for GitHub.com or "server" for GitHub Enterprise
    # host = "github.mycompany.com"  # Required when instance_type is "server"
  }
}
