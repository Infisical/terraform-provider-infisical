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

data "infisical_project_user" "example" {
  project_id = "<project-id>"
  user_id    = "<user-id>"
}

output "membership-id" {
  value = data.infisical_project_user.example.membership_id
}

output "username" {
  value = data.infisical_project_user.example.username
}

output "user" {
  value = data.infisical_project_user.example.user
}

output "roles" {
  value = data.infisical_project_user.example.roles
}
