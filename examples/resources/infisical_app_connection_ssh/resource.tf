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

resource "infisical_app_connection_ssh" "app-connection-ssh-password" {
  name   = "ssh-password-app-connection"
  method = "password"
  credentials = {
    host     = "server.example.com"
    port     = 22
    username = "<username>"
    password = "<password>"
  }
  # project_id   = "<project-id>" # Optional, only required if you want to scope the app connection to a specific project
  # gateway_id   = "<gateway-id>" # Optional, route through a specific Infisical Gateway instead of the Internet Gateway
  description = "I am a test app connection"
}

resource "infisical_app_connection_ssh" "app-connection-ssh-key" {
  name   = "ssh-key-app-connection"
  method = "ssh-key"
  credentials = {
    host        = "server.example.com"
    port        = 22
    username    = "<username>"
    private_key = "<ssh-private-key-pem>"
    # passphrase = "<private-key-passphrase>" # Optional, only if the private key is encrypted
  }
  # project_id   = "<project-id>" # Optional, only required if you want to scope the app connection to a specific project
  # gateway_id   = "<gateway-id>" # Optional, route through a specific Infisical Gateway instead of the Internet Gateway
  description = "I am a test app connection"
}
