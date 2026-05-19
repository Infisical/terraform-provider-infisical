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

resource "infisical_cert_manager_application" "platform" {
  name = "platform"
}

resource "infisical_cert_manager_application_group" "platform_sre_admins" {
  application_id = infisical_cert_manager_application.platform.id
  group_id       = "<group-id>"
  role           = "admin"
}

resource "infisical_cert_manager_application_group" "platform_devs" {
  application_id = infisical_cert_manager_application.platform.id
  group_id       = "<group-id>"
  role           = "operator"
}
