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

resource "infisical_app_connection_nutanix_prism_central" "app-connection-nutanix-api-key" {
  name   = "nutanix-api-key-app-connection"
  method = "api-key"
  credentials = {
    hostname                = "prism-central.example.com"
    api_key                 = "<32-character-lowercase-hex-api-key>"
    port                    = 9440
    ssl_reject_unauthorized = true
    # ssl_certificate       = file("ca.pem") # CA used to verify a self-signed TLS certificate
  }
  # project_id   = "<project-id>" # Optional, only required if you want to scope the app connection to a specific project
  # gateway_id   = "<gateway-id>" # Optional, route through a specific Infisical Gateway instead of the Internet Gateway
  description = "I am a test app connection"
}

resource "infisical_app_connection_nutanix_prism_central" "app-connection-nutanix-basic-auth" {
  name   = "nutanix-basic-auth-app-connection"
  method = "basic-auth"
  credentials = {
    hostname                = "prism-central.example.com"
    username                = "<username>"
    password                = "<password>"
    port                    = 9440
    ssl_reject_unauthorized = true
    # ssl_certificate       = file("ca.pem") # CA used to verify a self-signed TLS certificate
  }
  # project_id   = "<project-id>" # Optional, only required if you want to scope the app connection to a specific project
  # gateway_id   = "<gateway-id>" # Optional, route through a specific Infisical Gateway instead of the Internet Gateway
  description = "I am a test app connection"
}
