terraform {
  required_providers {
    infisical = {
      # version = <latest version>
      source = "infisical/infisical"
    }
  }
}

provider "infisical" {
  host = "http://app.infisical.com" # Only required if using self hosted instance of Infisical, default is https://app.infisical.com
  auth = {
    universal = {
      client_id     = "<machine-identity-client-id>"
      client_secret = "<machine-identity-client-secret>"
    }
  }
}

resource "infisical_app_connection_hashicorp_vault" "app-connection-vault-access-token" {
  name   = "vault-access-token-app-connection"
  method = "access-token"
  credentials = {
    instance_url = "https://vault.example.com"
    access_token = "<vault-access-token>"
    # namespace  = "<namespace>" # Optional, only for HCP Vault Dedicated/Enterprise
  }
  # project_id   = "<project-id>" # Optional, only required if you want to scope the app connection to a specific project
  description = "I am a test app connection"
}

resource "infisical_app_connection_hashicorp_vault" "app-connection-vault-app-role" {
  name   = "vault-app-role-app-connection"
  method = "app-role"
  credentials = {
    instance_url = "https://vault.example.com"
    role_id      = "<approle-role-id>"
    secret_id    = "<approle-secret-id>"
    # namespace  = "<namespace>" # Optional, only for HCP Vault Dedicated/Enterprise
  }
  # project_id   = "<project-id>" # Optional, only required if you want to scope the app connection to a specific project
  description = "I am a test app connection"
}
