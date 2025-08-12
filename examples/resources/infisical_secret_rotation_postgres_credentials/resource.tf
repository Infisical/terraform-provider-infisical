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

resource "infisical_secret_rotation_postgres_credentials" "postgres-credentials" {
  name          = "postgres-credentials-secret-rotation-example"
  project_id    = "<project-id>"
  environment   = "<environment-slug>"
  secret_path   = "<secret-path>" # Root folder is /
  connection_id = "<app-connection-id>"

  parameters = {
    username1 = "infisical_user_1"
    username2 = "infisical_user_2"
  }

  secrets_mapping = {
    username = "POSTGRES_DB_USERNAME"
    password = "POSTGRES_DB_PASSWORD"
  }
}
