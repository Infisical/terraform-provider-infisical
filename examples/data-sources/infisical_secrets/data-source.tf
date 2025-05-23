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
