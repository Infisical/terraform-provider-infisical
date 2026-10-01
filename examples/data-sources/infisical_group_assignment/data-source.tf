terraform {
  required_providers {
    infisical = {
      # version = <latest version>
      source = "infisical/infisical"
    }
  }
}

# Scope the provider to the sub-organization the group is linked into.
provider "infisical" {
  host = "https://app.infisical.com" # Only required if using self hosted instance of Infisical, default is https://app.infisical.com
  auth = {
    organization_slug = "<sub-organization-slug>"
    universal = {
      client_id     = "<machine-identity-client-id>"
      client_secret = "<machine-identity-client-secret>"
    }
  }
}

# Look up by slug (or use group_id instead).
data "infisical_group_assignment" "platform" {
  group_slug = "platform-engineering"
}

output "group_id" {
  value = data.infisical_group_assignment.platform.group_id
}

output "roles" {
  value = [for role in data.infisical_group_assignment.platform.roles : role.role_slug]
}
