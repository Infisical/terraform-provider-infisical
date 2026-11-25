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

variable "adcs_password" {
  type      = string
  sensitive = true
}

resource "infisical_app_connection_adcs" "adcs" {
  name   = "adcs-app-connection"
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
