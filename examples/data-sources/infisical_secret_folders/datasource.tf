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


data "infisical_secret_folders" "folders" {
  environment_slug = "dev"
  project_id       = "<PROJECT_ID>"
  folder_path      = "/"
}

output "secret-folders" {
  value = data.infisical_secret_folders.folders
}
