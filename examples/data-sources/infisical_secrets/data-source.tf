terraform {
  required_providers {
    infisical = {
      # version = <latest version>
      source = "hashicorp.com/edu/infisical"

    }
  }
}

provider "infisical" {
  host          = "https://app.infisical.com" # Only required if using self hosted instance of Infisical, default is https://app.infisical.com
  client_id     = "<>"
  client_secret = "<>"
}

data "infisical_secrets" "common-secrets" {
  env_slug     = "dev"
  workspace_id = "PROJECT_ID"
  folder_path  = "/some-folder/another-folder"
}

data "infisical_secrets" "backend-secrets" {
  env_slug     = "prod"
  workspace_id = "PROJECT_ID"
  folder_path  = "/"
}

output "all-project-secrets" {
  value = data.infisical_secrets.backend-secrets
}

output "single-secret" {
  value = data.infisical_secrets.backend-secrets.secrets["SECRET-NAME"]
}
