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

data "infisical_groups" "all" {}

resource "infisical_pam_group" "solengs" {
  group_id = one([for group in data.infisical_groups.all.groups : group.id if group.slug == "solengs"])
  role     = "member"
}
