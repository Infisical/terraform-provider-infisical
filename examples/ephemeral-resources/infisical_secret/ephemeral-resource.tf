terraform {
  required_providers {
    infisical = {
      # version = <latest version>
      source = "infisical/infisical"
    }
    postgresql = {
      source  = "cyrilgdn/postgresql"
      version = "1.25.0"
    }
  }
}

provider "infisical" {
  host          = "https://app.infisical.com" # Only required if using self hosted instance of Infisical, default is https://app.infisical.com
  client_id     = "<>"
  client_secret = "<>"
}

ephemeral "infisical_secret" "postgres_username" {
  name         = "POSTGRES_USERNAME"
  env_slug     = "dev"
  workspace_id = "PROJECT_ID"
  folder_path  = "/"
}

ephemeral "infisical_secret" "postgres_password" {
  name         = "POSTGRES_PASSWORD"
  env_slug     = "dev"
  workspace_id = "PROJECT_ID"
  folder_path  = "/"
}

locals {
  credentials = {
    username = ephemeral.infisical_secret.postgres_username.value
    password = ephemeral.infisical_secret.postgres_password.value
  }
}

provider "postgresql" {
  host     = data.aws_db_instance.example.address
  port     = data.aws_db_instance.example.port
  username = local.credentials["username"]
  password = local.credentials["password"]
}
