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

resource "infisical_cert_manager_application_user" "platform_admin" {
  application_id = infisical_cert_manager_application.platform.id
  email          = "admin@example.com"
  role           = "admin"
}

resource "infisical_cert_manager_application_user" "platform_operator" {
  application_id = infisical_cert_manager_application.platform.id
  email          = "operator@example.com"
  role           = "operator"
}

resource "infisical_cert_manager_application_user" "platform_auditor" {
  application_id = infisical_cert_manager_application.platform.id
  email          = "auditor@example.com"
  role           = "auditor"
}
