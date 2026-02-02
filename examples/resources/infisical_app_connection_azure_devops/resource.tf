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

resource "infisical_app_connection_azure_devops" "app_connection_azure_devops_client_secret" {
  name   = "app-connection-azure-devops-client-secret"
  method = "client-secret"
  credentials = {
    organization_name = "<azure-devops-organization-name>"
    tenant_id         = "<azure-tenant-id>"
    client_id         = "<azure-client-id>"
    client_secret     = "<azure-client-secret>"
  }
  description = "I am a test Azure DevOps app connection using client credentials"
}

resource "infisical_app_connection_azure_devops" "app_connection_azure_devops_access_token" {
  name   = "app-connection-azure-devops-access-token"
  method = "access-token"
  credentials = {
    organization_name = "<azure-devops-organization-name>"
    access_token      = "<azure-devops-access-token>"
  }
  description = "I am a test Azure DevOps app connection using access token"
}