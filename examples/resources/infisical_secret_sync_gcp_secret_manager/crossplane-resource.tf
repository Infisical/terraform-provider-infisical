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

resource "infisical_secret_sync_gcp_secret_manager" "secret_manager_test" {
  name          = "gcp-sync-tests"
  description   = "I am a test secret sync"
  project_id    = "f4517f4c-8b61-4727-8aef-5ae2807126fb"
  environment   = "prod"
  secret_path   = "/"
  connection_id = "<app-connection-id>"

  sync_options       = "{\"initial_sync_behavior\":\"import-prioritize-destination\"}"
  destination_config = "{\"project_id\":\"my-duplicate-project\"}"
}
