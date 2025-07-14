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

resource "infisical_secret_rotation_azure_client_secret" "azure-client-secret" {
  name          = "azure-client-secret-secret-rotation-example"
  project_id    = "<project-id>"
  environment   = "<environment-slug>"
  secret_path   = "<secret-path>" # Root folder is /
  connection_id = "<app-connection-id>"

  parameters = {
    object_id = "<azure-app-id>"
    client_id = "<azure-app-client-id>"
  }

  secrets_mapping = {
    client_id     = "AZURE_CLIENT_ID"
    client_secret = "AZURE_CLIENT_SECRET"
  }
}
