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

resource "infisical_secret_sync_azure_key_vault" "app-configuration-demo" {
  name          = "demo-sync"
  description   = "This is a demo sync."
  project_id    = "<project-id>"
  environment   = "dev"
  secret_path   = "/"
  connection_id = "<app-connection-id>" # The ID of your Azure App Connection

  sync_options = {
    initial_sync_behavior = "overwrite-destination"
  }
  destination_config = {
    vault_base_url = "<vault-base-url>", # https://example.vault.azure.net/
  }
}
