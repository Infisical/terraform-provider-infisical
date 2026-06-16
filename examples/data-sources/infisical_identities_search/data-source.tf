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

data "infisical_identities_search" "by_name_contains_both" {
  identity_name = "iac"
  mode          = "contains" # eq | contains
  scope         = "both"     # organization | project | both
}

output "identity_matches" {
  value = data.infisical_identities_search.by_name_contains_both.identities
}

output "total_count" {
  value = data.infisical_identities_search.by_name_contains_both.total_count
}

