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

resource "infisical_secret_sync_supabase" "example" {
  name          = "supabase-secret-sync"
  description   = "Sync secrets to Supabase project"
  project_id    = "<your-infisical-project-id>"
  connection_id = "<app-connection-id>" # The ID of your Supabase App Connection
  environment   = "<env-slug>"
  secret_path   = "<infisical-secret-path>"

  auto_sync_enabled = true

  destination_config = {
    project_id   = "<supabase-project-id>"
    project_name = "<supabase-project-name>"
  }

  sync_options = {
    initial_sync_behavior   = "overwrite-destination"
    disable_secret_deletion = false
    key_schema              = "{{secretKey}}-{{environment}}"
  }
}