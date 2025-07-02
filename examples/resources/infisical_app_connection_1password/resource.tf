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

resource "infisical_app_connection_1password" "one-password-demo" {
  name        = "1password-demo"
  description = "This is a demo 1password connection."
  method      = "api-token"
  credentials = {
    instance_url = "<https://1pass.example.com>"
    api_token    = "<API_TOKEN>"
  }
}
