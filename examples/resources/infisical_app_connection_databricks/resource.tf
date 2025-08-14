terraform {
  required_providers {
    infisical = {
      source = "infisical/infisical"
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