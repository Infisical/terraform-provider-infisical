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

# Example 1: Sync secrets to a GitLab project
resource "infisical_secret_sync_gitlab" "gitlab-project-sync" {
  name          = "gitlab-project-sync-demo"
  description   = "Demo of GitLab project secret sync"
  project_id    = "<project-id>"
  environment   = "<environment-slug>"
  secret_path   = "/" # Root folder is /
  connection_id = "<gitlab-app-connection-id>"

  sync_options = {
    initial_sync_behavior   = "overwrite-destination", # Supported options: overwrite-destination
    disable_secret_deletion = false,
    key_schema              = "INFISICAL_{{secretKey}}" # Optional, but recommended
  }

  destination_config = {
    scope                  = "project"               # Supported options: project|group
    project_id             = "<gitlab-project-id>"   # Required when scope is "project"
    project_name           = "<gitlab-project-name>" # Optional
    target_environment     = "*"                     # GitLab environment scope
    should_protect_secrets = false
    should_mask_secrets    = true
    should_hide_secrets    = false
  }
}

# Example 2: Sync secrets to a GitLab group
resource "infisical_secret_sync_gitlab" "gitlab-group-sync" {
  name          = "gitlab-group-sync-demo"
  description   = "Demo of GitLab group secret sync"
  project_id    = "<project-id>"
  environment   = "<environment-slug>"
  secret_path   = "/api"
  connection_id = "<gitlab-app-connection-id>"

  sync_options = {
    initial_sync_behavior   = "overwrite-destination",
    disable_secret_deletion = true,
    key_schema              = "{{secretKey}}"
  }

  destination_config = {
    scope                  = "group"               # Supported options: project|group
    group_id               = "<gitlab-group-id>"   # Required when scope is "group"
    group_name             = "<gitlab-group-name>" # Optional
    target_environment     = "production"          # GitLab environment scope
    should_protect_secrets = true
    should_mask_secrets    = true
    should_hide_secrets    = false
  }
}