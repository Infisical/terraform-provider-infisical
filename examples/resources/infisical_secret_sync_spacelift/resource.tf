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

resource "infisical_secret_sync_spacelift" "example" {
  name          = "spacelift-secret-sync"
  description   = "Sync secrets to a Spacelift context"
  project_id    = "<your-infisical-project-id>"
  connection_id = "<app-connection-id>" # The ID of your Spacelift App Connection
  environment   = "<env-slug>"
  secret_path   = "<infisical-secret-path>"

  auto_sync_enabled = true

  destination_config = {
    context_id   = "<spacelift-context-id>"
    context_name = "<spacelift-context-name>"
    config_type  = "environment-variable"
  }

  sync_options = {
    disable_secret_deletion = false
    key_schema              = "{{secretKey}}_{{environment}}"
    write_only              = true
  }
}
