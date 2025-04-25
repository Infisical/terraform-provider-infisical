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

resource "infisical_app_connection_aws" "app-connection-aws-assume-role" {
  name   = "aws-assume-role-app-connection"
  method = "assume-role"
  credentials = {
    role_arn = "<assume role arn>"
  }
  description = "I am a test app connection"
}

resource "infisical_app_connection_aws" "app-connection-aws-access-key" {
  name   = "aws-access-key-app-connection"
  method = "access-key"
  credentials = {
    access_key_id     = "<access-key-id>",
    secret_access_key = "<secret-access-key>",
  }
  description = "I am a test app connection"
}
