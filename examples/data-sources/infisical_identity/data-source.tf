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

# Look up an identity by ID.
data "infisical_identity" "by_id" {
  id = "<identity-id>"
}

# Or look up the same identity by name, for cases where the ID differs between
# environments but the name is kept consistent. Identity names are not unique, so
# this fails if more than one identity carries the name; the error lists the
# matching IDs so you can switch to `id` to pick one.
data "infisical_identity" "by_name" {
  name = "<identity-name>"
}

output "identity" {
  value = data.infisical_identity.by_id
}

output "identity_id_by_name" {
  value = data.infisical_identity.by_name.id
}
