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

resource "infisical_cert_manager_user" "admin" {
  email = "admin@example.com"
  role  = "admin"
}

resource "infisical_cert_manager_user" "oncall" {
  email = "oncall@example.com"
  role  = "member"
}
