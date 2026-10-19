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

resource "infisical_app_connection_spacelift" "example" {
  name        = "spacelift-connection"
  description = "I am a test app connection"
  method      = "api-key-secret"

  credentials = {
    api_url        = "https://<your-account>.app.spacelift.io"
    api_key_id     = "<your-spacelift-api-key-id>"
    api_key_secret = "<your-spacelift-api-key-secret>"
  }
}
