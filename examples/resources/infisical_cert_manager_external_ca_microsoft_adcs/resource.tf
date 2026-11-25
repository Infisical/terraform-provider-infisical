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

resource "infisical_app_connection_adcs" "adcs" {
  name   = "adcs-connection"
  method = "username-password"
  credentials = {
    host     = "ca01.corp.example.com"
    username = "CORP\\svc-infisical"
    password = var.adcs_password
  }
  gateway_id = "<gateway-id>"
}

resource "infisical_cert_manager_external_ca_microsoft_adcs" "adcs" {
  name   = "corporate-adcs"
  status = "active"

  app_connection_id = infisical_app_connection_adcs.adcs.id
  ca_name           = "corp-ca01-CA" # Optional, discovered from the CA host over SMB (TCP 445) when omitted
}
