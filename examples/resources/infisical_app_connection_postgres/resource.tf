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

resource "infisical_app_connection_postgres" "postgres-demo" {
  name        = "postgres-demo"
  description = "This is a demo postgres connection."
  method      = "username-and-password"
  credentials = {
    host        = "example.com"
    port        = 5432
    database    = "default"
    username    = "postgres"
    password    = "<password>"
    ssl_enabled = false
  }
}
