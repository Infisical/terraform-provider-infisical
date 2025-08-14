terraform {
  required_providers {
    infisical = {
      source = "infisical/infisical"
    }
  }
}

resource "infisical_secret_sync_bitbucket" "example" {
  name          = "bitbucket-secret-sync"
  description   = "Sync secrets to Bitbucket repository"
  project_id    = "your-infisical-project-id"
  connection_id = infisical_app_connection_bitbucket.example.id
  environment   = "dev"
  secret_path   = "/app"

  auto_sync_enabled = true

  destination_config = {
    repository_slug = "my-repository"
    workspace_slug  = "my-workspace"
    environment_id  = "production"
  }

  sync_options = {
    initial_sync_behavior   = "overwrite-destination"
    disable_secret_deletion = false
    key_schema              = "{{secretKey}}-{{environment}}"
  }
}