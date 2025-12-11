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

resource "infisical_app_connection_azure_client_secrets" "app_connection_azure_client_secret" {
  name   = "app_connection_azure_client_secret"
  method = "client-secret"
  credentials = {
    tenant_id     = "<azure-tenant-id>"
    client_id     = "<azure-client-id>"
    client_secret = "<azure-client-secret>"
  }
  description = "I am a test Azure app connection using client credentials"
}
