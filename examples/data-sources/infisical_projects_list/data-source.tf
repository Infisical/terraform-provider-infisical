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

# Fetch specific projects by slug
data "infisical_projects_list" "filtered" {
  slugs = ["<project-slug-1>", "<project-slug-2>"]
}

# Fetch all projects the machine identity has access to (omit slugs)
data "infisical_projects_list" "all" {}

# Get the filtered projects
output "filtered-projects" {
  value = data.infisical_projects_list.filtered.projects
}

# Get all accessible projects
output "all-projects" {
  value = data.infisical_projects_list.all.projects
}
