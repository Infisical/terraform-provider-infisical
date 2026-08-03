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

locals {
  requested_slugs = ["<project-slug-1>", "<project-slug-2>", "<project-slug-3>"]
}

# Fetch specific projects by slug
data "infisical_projects_list" "filtered" {
  slugs = local.requested_slugs
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

# Create any requested projects that do not already exist.
# Slugs not found are omitted from the data source result.
locals {
  existing_slugs = [for p in data.infisical_projects_list.filtered.projects : p.slug]
  missing_slugs  = setsubtract(local.requested_slugs, local.existing_slugs)
}

resource "infisical_project" "missing" {
  for_each = toset(local.missing_slugs)
  name     = each.key
  slug     = each.key
}
