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

resource "infisical_cert_manager_group" "sre" {
  group_id = "<group-id>"
  role     = "admin"
}

resource "infisical_cert_manager_group" "oncall_rotation" {
  group_id = "<group-id>"
  role     = "member"
}
