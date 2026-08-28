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

// Look up a project by its slug.
data "infisical_projects" "by-slug" {
  slug = "<project-slug>"
}

// Or look up the same project by its ID, for cases where the ID is what you already
// have on hand. Set exactly one of slug or id; setting both, or neither, is an error.
data "infisical_projects" "by-id" {
  id = "<project-id>"
}

// Get the value of the "dev" environment
output "dev-environment" {
  value = data.infisical_projects.by-slug.environments["dev"]
}

// Get the entire project
output "entire-project" {
  value = data.infisical_projects.by-slug
}

// The two lookups resolve to the same project
output "project-slug-by-id" {
  value = data.infisical_projects.by-id.slug
}
