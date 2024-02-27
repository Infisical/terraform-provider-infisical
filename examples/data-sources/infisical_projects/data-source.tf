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
  client_id     = "<machine-identity-client-id>"
  client_secret = "<machine-identity-client-secret>"
}

data "infisical_projects" "test-project" {
  slug = "new-test"
}

// Get the value of the "dev" environment
output "dev-environment" {
  value = data.infisical_projects.test-project.environments["dev"]
}

// Get the entire project
output "entire-project" {
  value = data.infisical_projects.test-project
}
