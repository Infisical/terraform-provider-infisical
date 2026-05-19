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

resource "infisical_cert_manager_application_identity" "platform_ci" {
  application_id = infisical_cert_manager_application.platform.id
  identity_id    = "<identity-id>"
  role           = "operator"
}

resource "infisical_cert_manager_application_identity" "platform_readonly" {
  application_id = infisical_cert_manager_application.platform.id
  identity_id    = "<identity-id>"
  role           = "auditor"
}
