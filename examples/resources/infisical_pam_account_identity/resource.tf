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

resource "infisical_pam_identity" "ci" {
  identity_id = "<identity-id>"
  role        = "member"
}

resource "infisical_pam_account_identity" "ci_orders" {
  account_id  = "<account-id>"
  identity_id = infisical_pam_identity.ci.identity_id # also makes Terraform add the identity to PAM first
  role        = "connector"
}
