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

# List all the identities in the organization. Scope defaults to ["organization", "project"].
data "infisical_identities_list" "org" {}

# List the identities that projects own, for the projects that the caller can access.
data "infisical_identities_list" "project" {
  scope = ["project"]
}

# List both organization and project identities.
data "infisical_identities_list" "all" {
  scope = ["organization", "project"]

  filter = {
    identity_names = ["kubernetes-identity"]
  }
}

output "org_identity_names" {
  value = [for identity in data.infisical_identities_list.org.identities : identity.name]
}

# Find the identities that have Universal Auth enabled.
output "identities_with_universal_auth" {
  value = [
    for identity in data.infisical_identities_list.all.identities : identity.id
    if contains(identity.auth_modes, "universal-auth")
  ]
}
