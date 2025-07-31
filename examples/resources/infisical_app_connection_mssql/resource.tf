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

resource "infisical_app_connection_mssql" "mssql-demo" {
  name        = "mssql-demo"
  description = "This is a demo mssql connection."
  method      = "username-and-password"
  credentials = {
    host        = "example.com"
    port        = 1433
    database    = "default"
    username    = "root"
    password    = "<password>"
    ssl_enabled = false
  }
}
