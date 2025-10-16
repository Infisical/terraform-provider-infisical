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

resource "infisical_secret_folder" "folder-1" {
  name             = "folder-1"
  environment_slug = "dev"
  project_id       = "<PROJECT-ID>"
  folder_path      = "/"
  # force_delete     = true
}

resource "infisical_secret_folder" "folder-2" {
  name             = "folder-2"
  environment_slug = "prod"
  project_id       = "<PROJECT-ID>"
  folder_path      = "/nested"
  # force_delete     = false
}

