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

resource "infisical_app_connection_databricks" "example" {
  name        = "databricks-connection"
  description = "Databricks connection for secret sync"
  method      = "service-principal"

  credentials = {
    client_id     = "your-databricks-client-id"
    client_secret = "your-databricks-client-secret"
    workspace_url = "https://your-workspace.cloud.databricks.com"
  }
}
