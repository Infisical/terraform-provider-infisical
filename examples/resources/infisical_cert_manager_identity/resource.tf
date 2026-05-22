terraform {
  required_providers {
    infisical = {
      source = "infisical/infisical"
    }
  }
}

provider "infisical" {
  host          = "https://app.infisical.com"
  client_id     = var.client_id
  client_secret = var.client_secret
}

resource "infisical_cert_manager_identity" "ci" {
  identity_id = "<identity-id>"
  role        = "admin"
}

resource "infisical_cert_manager_identity" "deploy_bot" {
  identity_id = "<identity-id>"
  role        = "member"
}
