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

data "infisical_secret_metadata" "example" {
  name         = "MY_SECRET"
  env_slug     = "dev"
  workspace_id = "<project-id>"
  folder_path  = "/"
}

output "secret_version" {
  value = data.infisical_secret_metadata.example.secret_version
}

output "secret_type" {
  value = data.infisical_secret_metadata.example.type
}
