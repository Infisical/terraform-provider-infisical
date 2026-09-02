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

resource "infisical_app_connection_digicert" "digicert" {
  name   = "digicert-connection"
  method = "api-key"
  credentials = {
    api_key = var.digicert_api_key
    region  = "us"
  }
}

# Issues SSL/TLS certificates
resource "infisical_cert_manager_external_ca_digicert" "ssl" {
  name   = "digicert-ssl"
  status = "active"

  app_connection_id = infisical_app_connection_digicert.digicert.id
  organization_id   = 123456
  product_name_id   = "ssl_plus"
  purpose           = "ssl"
}

# Issues code signing certificates, which require a verified contact
resource "infisical_cert_manager_external_ca_digicert" "code_signing" {
  name   = "digicert-code-signing"
  status = "active"

  app_connection_id = infisical_app_connection_digicert.digicert.id
  organization_id   = 123456
  product_name_id   = "code_signing"
  purpose           = "code_signing"

  verified_contact = {
    first_name = "Jane"
    last_name  = "Doe"
    email      = "jane.doe@example.com"
    job_title  = "Security Engineer"
    telephone  = "+1-555-000-0000"
  }
}
