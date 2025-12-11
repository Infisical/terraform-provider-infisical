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

resource "infisical_app_connection_cloudflare" "app-connection-cloudflare" {
  name   = "cloudflare-app-connection"
  method = "api-token"
  credentials = {
    account_id = "<cloudflare-account-id>"
    api_token  = "<cloudflare-api-token>"
  }
  description = "I am a Cloudflare app connection"
}