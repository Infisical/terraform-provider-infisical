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

resource "infisical_secret_sync_cloudflare_workers" "cloudflare-workers-secret-sync" {
  name          = "cloudflare-workers-secret-sync-demo"
  description   = "Demo of Cloudflare Workers secret sync"
  project_id    = "<project-id>"
  environment   = "<environment-slug>"
  secret_path   = "<secret-path>" # Root folder is /
  connection_id = "<cloudflare-app-connection-id>"

  sync_options = {
    initial_sync_behavior   = "overwrite-destination" # Supported options: overwrite-destination, import-prioritize-source, import-prioritize-destination
    disable_secret_deletion = false
    key_schema              = "<key-schema>" # Optional: The format to use for structuring secret keys
  }

  destination_config = {
    script_id = "<cloudflare-workers-script-id>"
  }
}