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

resource "infisical_app_connection_digicert" "digicert-demo" {
  name        = "digicert-demo"
  description = "This is a demo DigiCert connection."
  method      = "api-key"
  credentials = {
    api_key = "<API_KEY>"
    region  = "us"
  }
}
