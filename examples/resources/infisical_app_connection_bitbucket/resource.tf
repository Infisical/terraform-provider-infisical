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

resource "infisical_app_connection_bitbucket" "example" {
  name        = "bitbucket-connection"
  description = "Bitbucket connection for secret sync"
  method      = "api-token"

  credentials = {
    email     = "your-bitbucket-email@example.com"
    api_token = "your-bitbucket-api-token"
  }
}