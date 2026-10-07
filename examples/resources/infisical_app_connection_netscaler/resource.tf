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

resource "infisical_app_connection_netscaler" "app-connection-netscaler" {
  name   = "netscaler-app-connection"
  method = "basic-auth"
  credentials = {
    hostname                = "netscaler.example.com"
    username                = "<username>"
    password                = "<password>"
    port                    = 443
    ssl_reject_unauthorized = true
    # ssl_certificate       = file("ca.pem") # CA used to verify a self-signed TLS certificate
  }
  # project_id   = "<project-id>" # Optional, only required if you want to scope the app connection to a specific project
  # gateway_id   = "<gateway-id>" # Optional, route through a specific Infisical Gateway instead of the Internet Gateway
  description = "I am a test app connection"
}
