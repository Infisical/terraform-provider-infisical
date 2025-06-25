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

resource "infisical_secret_sync_github" "example-github-secret-sync" {
  name          = "github-secret-sync-demo"
  description   = "Demo of Github secret sync"
  project_id    = "<project-id>"
  environment   = "<environment-slug>"
  secret_path   = "/" # Root folder is /
  connection_id = "<github-app-connection-id>"

  sync_options = {
    initial_sync_behavior   = "overwrite-destination", # Supported options: overwrite-destination
    disable_secret_deletion = false,
    key_schema              = "INFISICAL_{{secretKey}}" # Optional, but recommended
  }

  destination_config = {
    scope            = "repository"                # Supported options: repository|organization|repository-environment
    repository_owner = "<github-repository-owner>" # The github organization name or github username for personal repositories
    repository_name  = "<github-repository-name>"  # The github repository name
  }
}
