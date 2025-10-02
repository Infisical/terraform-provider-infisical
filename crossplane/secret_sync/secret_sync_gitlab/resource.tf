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

  sync_options = "{\"initial_sync_behavior\":\"overwrite-destination\",\"disable_secret_deletion\":false,\"key_schema\":\"INFISICAL_{{secretKey}}\"}"

  destination_config = "{\"scope\":\"project\",\"project_id\":\"<gitlab-project-id>\",\"project_name\":\"<gitlab-project-name>\",\"target_environment\":\"*\",\"should_protect_secrets\":false,\"should_mask_secrets\":true,\"should_hide_secrets\":false}"
}

# Example 2: Sync secrets to a GitLab group
resource "infisical_secret_sync_gitlab" "gitlab-group-sync" {
  name          = "gitlab-group-sync-demo"
  description   = "Demo of GitLab group secret sync"
  project_id    = "<project-id>"
  environment   = "<environment-slug>"
  secret_path   = "/"
  connection_id = "<gitlab-app-connection-id>"

  sync_options = "{\"initial_sync_behavior\":\"overwrite-destination\",\"disable_secret_deletion\":true,\"key_schema\":\"{{secretKey}}\"}"

  destination_config = "{\"scope\":\"group\",\"group_id\":\"<gitlab-group-id>\",\"group_name\":\"<gitlab-group-name>\",\"target_environment\":\"production\",\"should_protect_secrets\":true,\"should_mask_secrets\":true,\"should_hide_secrets\":false}"
}
