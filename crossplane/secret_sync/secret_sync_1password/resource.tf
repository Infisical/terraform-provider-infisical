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



resource "infisical_secret_sync_1password" "one-password-secret-sync-demo" {
  name          = "1password-secret-sync-demo"
  description   = "This is a demo 1Password Secret Sync."
  project_id    = "<project-id>"
  environment   = "<environment-slug>"
  secret_path   = "<secret-path>"
  connection_id = "<app-connection-id>"

  sync_options = "{\"initialSyncBehavior\": \"<initial-sync-behavior>\", \"keySchema\": \"<key-schema>\"}"

  destination_config = "{\"vaultId\": \"<vault-id>\", \"valueLabel\": \"<value-label>\"}"
}
