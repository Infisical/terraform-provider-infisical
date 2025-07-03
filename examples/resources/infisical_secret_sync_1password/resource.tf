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

resource "infisical_app_connection_1password" "one-password-app-connection-demo" {
  name        = "1password-app-connection-demo"
  description = "This is a demo 1Password App Connection."
  method      = "api-token"
  credentials = {
    instance_url = "<https://1pass.example.com>"
    api_token    = "<API_TOKEN>"
  }
}

resource "infisical_secret_sync_1password" "one-password-secret-sync-demo" {
  name          = "1password-secret-sync-demo"
  description   = "This is a demo 1Password Secret Sync."
  project_id    = "<project-id>"
  environment   = "<environment-slug>"
  secret_path   = "<secret-path>"
  connection_id = infisical_app_connection_1password.one-password-app-connection-demo.id
  destination_config = {
    vault_id    = "<vault-id>"
    value_label = "<value-label>" # Optional, defaults to `value`
  }
  sync_options = {
    initial_sync_behavior = "<initial-sync-behavior>" # Supported options: overwrite-destination|import-prioritize-source|import-prioritize-destination
    key_schema            = "<key-schema>"            // Optional
  }
}
