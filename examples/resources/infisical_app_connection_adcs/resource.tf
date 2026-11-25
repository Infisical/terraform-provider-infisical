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

resource "infisical_app_connection_adcs" "app-connection-adcs" {
  name   = "adcs-app-connection"
  method = "username-password"
  credentials = {
    host     = "ca01.corp.example.com"
    username = "CORP\\svc-infisical"
    password = "<password>"
  }
  # project_id   = "<project-id>" # Optional, only required if you want to scope the app connection to a specific project
  gateway_id  = "<gateway-id>" # Required, Microsoft ADCS connections reach the AD CS server through a gateway
  description = "I am a test app connection"
}
