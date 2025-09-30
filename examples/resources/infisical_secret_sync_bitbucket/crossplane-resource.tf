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

resource "infisical_secret_sync_bitbucket" "example" {
  name          = "bitbucket-secret-sync"
  description   = "Sync secrets to Bitbucket repository"
  project_id    = "<your-infisical-project-id>"
  connection_id = "<app-connection-id>" # The ID of your Bitbucket App Connection
  environment   = "<env-slug>"
  secret_path   = "<infisical-secret-path>"

  auto_sync_enabled = true


  sync_options = "{\"initial_sync_behavior\":\"overwrite-destination\",\"disable_secret_deletion\":false,\"key_schema\":\"{{secretKey}}-{{environment}}\"}"

  destination_config = "{\"repository_slug\":\"<bitbucket-repository-slug>\",\"workspace_slug\":\"<bitbucket-workspace-slug>\",\"environment_id\":\"<bitbucket-environment-slug>\"}"
}
