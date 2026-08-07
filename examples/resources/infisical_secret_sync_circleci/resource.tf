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

resource "infisical_secret_sync_circleci" "example" {
  name          = "circleci-secret-sync"
  description   = "Sync secrets to a CircleCI project"
  project_id    = "<your-infisical-project-id>"
  connection_id = "<app-connection-id>" # The ID of your CircleCI App Connection
  environment   = "<env-slug>"
  secret_path   = "<infisical-secret-path>"

  auto_sync_enabled = true

  destination_config = {
    org_name     = "<circleci-organization-name>"
    project_name = "<circleci-project-name>"
    project_id   = "<circleci-project-id>"
  }

  sync_options = {
    initial_sync_behavior   = "overwrite-destination"
    disable_secret_deletion = false
    key_schema              = "{{secretKey}}-{{environment}}"
  }
}
