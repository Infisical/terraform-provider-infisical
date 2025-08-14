terraform {
  required_providers {
    infisical = {
      source = "infisical/infisical"
    }
  }
}

resource "infisical_secret_sync_databricks" "example" {
  name          = "databricks-secret-sync"
  description   = "Sync secrets to Databricks secret scope"
  project_id    = "your-infisical-project-id"
  connection_id = infisical_app_connection_databricks.example.id
  environment   = "dev"
  secret_path   = "/app"

  auto_sync_enabled = true

  destination_config = {
    scope = "infisical-secrets"
  }

  sync_options = {
    initial_sync_behavior   = "overwrite-destination"
    disable_secret_deletion = false
    key_schema              = "{{secretKey}}-{{environment}}"
  }
}
