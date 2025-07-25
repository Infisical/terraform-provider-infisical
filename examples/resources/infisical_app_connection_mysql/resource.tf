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

resource "infisical_app_connection_mysql" "mysql-demo" {
  name        = "mysql-demo"
  description = "This is a demo mysql connection."
  method      = "username-and-password"
  credentials = {
    host        = "example.com"
    port        = 3306
    database    = "default"
    username    = "root"
    password    = "<password>"
    ssl_enabled = false
  }
}
