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

data "infisical_secret_properties" "example" {
  name             = "MY_SECRET"
  environment_slug = "<environment-slug>"
  project_id       = "<project-id>"
  folder_path      = "<folder-path>"
}

output "secret_version" {
  value = data.infisical_secret_properties.example.secret_version
}

output "secret_type" {
  value = data.infisical_secret_properties.example.secret_type
}

output "tags" {
  value = data.infisical_secret_properties.example.tags
}

output "secret_metadata" {
  value = data.infisical_secret_properties.example.secret_metadata
}
