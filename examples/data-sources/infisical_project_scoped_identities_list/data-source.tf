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

# Fetch all project-scoped machine identities in a project
data "infisical_project_scoped_identities_list" "all" {
  project_id = "<project-id>"
}

# Fetch specific identities by name
data "infisical_project_scoped_identities_list" "filtered" {
  project_id = "<project-id>"
  names      = ["my-project-eso", "my-project-ci"]
}

# Output all identities
output "all-identities" {
  value = data.infisical_project_scoped_identities_list.all.identities
}

# Output filtered identities
output "filtered-identities" {
  value = data.infisical_project_scoped_identities_list.filtered.identities
}
