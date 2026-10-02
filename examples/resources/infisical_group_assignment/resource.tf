terraform {
  required_providers {
    infisical = {
      # version = <latest version>
      source = "infisical/infisical"
    }
  }
}

# Infisical links the group into whichever organization the session is scoped to, so this
# provider must be scoped to the target SUB-ORGANIZATION through auth.organization_slug.
# The machine identity must have access to that sub-organization (for example, the identity
# that created it with infisical_sub_organization is added to it as an admin).
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

# Link a root-organization group into the sub-organization by slug...
resource "infisical_group_assignment" "platform" {
  group_slug = "platform-engineering"
  roles = [
    {
      role_slug = "member"
    },
  ]
}

# ...or by ID, with a custom sub-organization role and a temporary admin role.
resource "infisical_group_assignment" "security" {
  group_id = "<root-organization-group-id>"
  roles = [
    {
      role_slug = "my-custom-org-role"
    },
    {
      role_slug                   = "admin"
      is_temporary                = true
      temporary_access_start_time = "2026-10-01T09:00:00Z"
      temporary_range             = "7d"
    },
  ]
}

# The root-organization group can be created in the same workspace with a second provider
# instance scoped to the root organization:
#
#   provider "infisical" {
#     alias = "root"
#     auth  = { universal = { client_id = "...", client_secret = "..." } }
#   }
#
#   resource "infisical_group" "platform" {
#     provider = infisical.root
#     name     = "Platform Engineering"
#     slug     = "platform-engineering"
#     role     = "member"
#   }
#
#   resource "infisical_group_assignment" "platform" {
#     group_id = infisical_group.platform.id
#     roles    = [{ role_slug = "member" }]
#   }
#
# The sub-organization must already exist when this provider is configured. See the
# infisical_sub_organization documentation for the two-configuration pattern.
