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

resource "infisical_secret" "mongo_secret" {
  name         = "MONGO_DB"
  value        = "<some-key>"
  env_slug     = "dev"
  workspace_id = "PROJECT_ID"
  folder_path  = "/"
}

resource "infisical_secret" "smtp_secret" {
  name         = "SMTP"
  value        = "<some key>"
  env_slug     = "dev"
  workspace_id = "PROJECT_ID"
  folder_path  = "/mail-service"
  secret_reminder = {
    note        = "Rotate this secret using X API"
    repeat_days = 30
  }
}


resource "infisical_secret_tag" "terraform" {
  name       = "terraform"
  slug       = "terraform"
  color      = "#fff"
  project_id = "PROJECT_ID"
}

resource "infisical_secret" "github_action_secret" {
  name         = "GITHUB_ACTION"
  value        = "<some value>"
  env_slug     = "dev"
  workspace_id = "PROJECT_ID"
  folder_path  = "/"
  tag_ids      = [infisical_secret_tag.terraform.id]
}

# Ephemeral resource (requires Terraform 1.10.0+)
# https://www.hashicorp.com/blog/terraform-1-10-improves-handling-secrets-in-state-with-ephemeral-values
ephemeral "infisical_secret" "ephemeral-secret" {
  name         = "SECRET-KEY"
  env_slug     = "dev"
  workspace_id = "PROJECT_ID"
  folder_path  = "/"
}
