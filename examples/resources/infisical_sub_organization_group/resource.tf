terraform {
  required_providers {
    infisical = {
      # version = <latest version>
      source = "infisical/infisical"
    }
  }
}

# Groups get linked into whatever org the provider is scoped to, so point it at the
# sub-org with auth.organization_slug.
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

# Link a root-organization group by slug
resource "infisical_sub_organization_group" "platform" {
  group_slug = "platform-engineering"
  roles = [
    {
      role_slug = "member"
    }
  ]
}

# Link a root-organization group by ID, with a custom role and a temporary admin role
resource "infisical_sub_organization_group" "security" {
  group_id = "<root-organization-group-id>"
  roles = [
    {
      role_slug = "my-custom-org-role"
    },
    {
      role_slug                   = "admin"
      is_temporary                = true
      temporary_range             = "7d"
      temporary_access_start_time = "2026-10-01T09:00:00Z"
    }
  ]
}

# Link many root-organization groups at once
variable "linked_groups" {
  type = map(string)
  default = {
    "developers" = "member"
    "auditors"   = "no-access"
  }
}

resource "infisical_sub_organization_group" "linked" {
  for_each   = var.linked_groups
  group_slug = each.key
  roles = [
    {
      role_slug = each.value
    }
  ]
}
