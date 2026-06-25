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

data "infisical_projects" "multiple-projects" {
  slugs = ["first-project-slug", "second-project-slug"]
}

// Get a specific project by its slug
output "first-project" {
  value = data.infisical_projects.multiple-projects.projects["first-project-slug"]
}

// Get the value of the "dev" environment of a specific project
output "first-project-dev-environment" {
  value = data.infisical_projects.multiple-projects.projects["first-project-slug"].environments["dev"]
}

// Get all fetched projects (map keyed by slug)
output "all-projects" {
  value = data.infisical_projects.multiple-projects.projects
}
