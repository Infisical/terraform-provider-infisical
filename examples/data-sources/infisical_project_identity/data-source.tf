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

data "infisical_project_identity" "example" {
  project_id  = "<project-id>"
  identity_id = "<identity-id>"
}

output "membership-id" {
  value = data.infisical_project_identity.example.membership_id
}

# All roles assigned to the identity
output "roles" {
  value = data.infisical_project_identity.example.roles
}
