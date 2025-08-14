terraform {
  required_providers {
    infisical = {
      source = "infisical/infisical"
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