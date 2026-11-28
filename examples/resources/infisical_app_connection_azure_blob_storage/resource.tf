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

resource "infisical_app_connection_azure_blob_storage" "client_secret" {
  name   = "pam-recordings-azure"
  method = "client-secret"
  credentials = {
    account_name  = "acmepamrecordings"
    tenant_id     = "<azure-tenant-id>"
    client_id     = "<azure-client-id>"
    client_secret = "<azure-client-secret>"
  }
  description = "Stores PAM session recordings in Azure Blob Storage"
}

resource "infisical_app_connection_azure_blob_storage" "access_key" {
  name   = "pam-recordings-azure-key"
  method = "access-key"
  credentials = {
    account_name = "acmepamrecordings"
    account_key  = "<storage-account-key>"
  }
}

resource "infisical_pam_account_template" "windows" {
  name         = "windows-azure-recordings"
  account_type = "windows"

  settings = {
    recording = {
      storage_backend = "azure-blob"
      connection_id   = infisical_app_connection_azure_blob_storage.client_secret.id
      azure_blob = {
        container  = "pam-recordings"
        key_prefix = "windows/"
      }
    }
  }
}
