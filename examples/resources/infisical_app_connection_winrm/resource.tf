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

resource "infisical_app_connection_winrm" "app-connection-winrm" {
  name   = "winrm-app-connection"
  method = "username-password"
  credentials = {
    host                    = "windows-host.example.com"
    port                    = 5985
    username                = "EXAMPLE\\administrator"
    password                = "<password>"
    ssl_enabled             = true
    ssl_reject_unauthorized = true
    # ssl_certificate       = file("ca.pem") # CA used to verify a self-signed HTTPS listener
  }
  # project_id   = "<project-id>" # Optional, only required if you want to scope the app connection to a specific project
  # gateway_id   = "<gateway-id>" # Optional, route through a specific Infisical Gateway instead of the Internet Gateway
  description = "I am a test app connection"
}
