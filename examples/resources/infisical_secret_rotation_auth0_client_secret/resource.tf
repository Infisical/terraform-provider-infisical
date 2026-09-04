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

resource "infisical_secret_rotation_auth0_client_secret" "auth0-client-secret" {
  name          = "auth0-client-secret-secret-rotation-example"
  project_id    = "<project-id>"
  environment   = "<environment-slug>"
  secret_path   = "<secret-path>" # Root folder is /
  connection_id = "<app-connection-id>"

  parameters = {
    client_id = "<auth0-application-client-id>"
  }

  secrets_mapping = {
    client_id     = "AUTH0_CLIENT_ID"
    client_secret = "AUTH0_CLIENT_SECRET"
  }
}
