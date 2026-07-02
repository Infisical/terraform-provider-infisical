terraform {
  required_providers {
    infisical = {
      # version = <latest version>
      source = "infisical/infisical"
    }
  }
}

provider "infisical" {
  host = "http://app.infisical.com" # Only required if using self hosted instance of Infisical, default is https://app.infisical.com
  auth = {
    universal = {
      client_id     = "<machine-identity-client-id>"
      client_secret = "<machine-identity-client-secret>"
    }
  }
}

# The authenticating machine identity must belong to the root organization and have
# the "sub-organization" create permission. The identity used to create the sub-org
# is automatically added to it as an admin.
resource "infisical_sub_organization" "example" {
  name = "Engineering"
  slug = "engineering" # Optional. If omitted, Infisical generates one from the name.
}

# To manage resources INSIDE the sub-organization (identities, groups, custom roles),
# configure a second, aliased provider scoped to the sub-org via organization_slug:
#
# provider "infisical" {
#   alias = "suborg"
#   host  = "http://app.infisical.com"
#   auth = {
#     organization_slug = infisical_sub_organization.example.slug
#     universal = {
#       client_id     = "<machine-identity-client-id>"
#       client_secret = "<machine-identity-client-secret>"
#     }
#   }
# }
