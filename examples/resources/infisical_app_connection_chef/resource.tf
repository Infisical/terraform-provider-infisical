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

resource "infisical_app_connection_chef" "app-connection-chef" {
  name   = "chef-app-connection"
  method = "user-key"
  credentials = {
    org_name    = "<chef-organization-name>"
    user_name   = "<chef-user-name>"
    private_key = "<chef-user-private-key-pem>"
    # server_url = "https://chef.example.com" # Optional, defaults to the hosted Chef API
  }
  # project_id   = "<project-id>" # Optional, only required if you want to scope the app connection to a specific project
  # gateway_id   = "<gateway-id>" # Optional, route through a specific Infisical Gateway instead of the Internet Gateway
  description = "I am a test app connection"
}
