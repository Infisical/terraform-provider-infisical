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

resource "infisical_pam_group" "solengs" {
  group_id = "<group-id>"
  role     = "member"
}

resource "infisical_pam_account_group" "solengs_pamadmin" {
  account_id = "<account-id>"
  group_id   = infisical_pam_group.solengs.group_id # also makes Terraform add the group to PAM first
  role       = "operator"
}
