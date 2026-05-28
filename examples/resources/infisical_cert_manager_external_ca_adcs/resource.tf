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

resource "infisical_cert_manager_external_ca_adcs" "adcs" {
  name   = "corporate-adcs"
  status = "active"

  azure_adcs_connection_id = "your-azure-adcs-connection-id"
}
