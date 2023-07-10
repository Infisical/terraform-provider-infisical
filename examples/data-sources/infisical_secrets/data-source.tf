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
  service_token = "<>"
}

data "infisical_secrets" "common-secrets" {
  env_slug    = "dev"
  folder_path = "/some-folder/another-folder"
}

data "infisical_secrets" "backend-secrets" {
  env_slug    = "prod"
  folder_path = "/"
}

output "all-project-secrets" {
  value = data.infisical_secrets.backend-secrets
}

output "single-secret" {
  value = data.infisical_secrets.backend-secrets.secrets["SECRET-NAME"]
}
