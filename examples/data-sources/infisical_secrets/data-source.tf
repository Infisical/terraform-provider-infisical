terraform {
  required_providers {
    infisical = {
      # version = <latest version>
      source = "infisical/infisical"
    }
  }
}

provider "infisical" {
  host          = "https://app.infisical.com" # Only required if using self hosted instance of Infisical, default is https://app.infisical.com
  client_id     = "<universal-auth-client-id>"
  client_secret = "<universal-auth-client-secret>"
}

data "infisical_secrets" "common_secrets" {
  env_slug     = "dev"
  workspace_id = "<project id>" // project ID
  folder_path  = "/"
}

output "all-project-secrets" {
  value = nonsensitive(data.infisical_secrets.common_secrets.secrets["SECRET-NAME"].value)
}

output "all-project-secrets" {
  value = nonsensitive(data.infisical_secrets.common_secrets.secrets["SECRET-NAME"].comment)
}
