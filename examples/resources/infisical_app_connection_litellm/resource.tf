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

resource "infisical_app_connection_litellm" "litellm-demo" {
  name        = "litellm-demo"
  description = "This is a demo LiteLLM connection."
  method      = "api-key"
  credentials = {
    instance_url = "https://litellm.example.com"
    api_key      = "<LITELLM_API_KEY>"
  }
}
