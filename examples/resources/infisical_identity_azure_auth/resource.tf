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

resource "infisical_identity" "machine-identity-1" {
  name   = "machine-identity-1"
  role   = "admin"
  org_id = "601815be-6884-4ee4-86c7-bfc6415f2123"
}

resource "infisical_identity_azure_auth" "azure-auth" {
  identity_id                   = infisical_identity.machine-identity-1.id
  tenant_id                     = "<>"
  resource_url                  = "https://management.azure.com/"
  allowed_service_principal_ids = ["<>", "<>"]
  access_token_ttl              = 2592000
  access_token_max_ttl          = 2592000
}
