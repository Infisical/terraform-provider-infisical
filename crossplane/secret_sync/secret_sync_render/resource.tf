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


resource "infisical_secret_sync_render" "render-secret-sync-demo" {
  name          = "render-secret-sync-demo"
  description   = "This is a demo Render Secret Sync."
  project_id    = "<project-id>"
  environment   = "<environment-slug>"
  secret_path   = "<secret-path>"
  connection_id = "<app-connection-id>"

  sync_options = "{\"initial_sync_behavior\":\"<initial-sync-behavior>\",\"key_schema\":\"<key-schema>\"}"

  destination_config = "{\"service_id\":\"<service-id>\",\"scope\":\"<scope>\",\"type\":\"<type>\"}"
}
