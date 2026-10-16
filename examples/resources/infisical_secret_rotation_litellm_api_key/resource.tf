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

resource "infisical_secret_rotation_litellm_api_key" "litellm-api-key" {
  name          = "litellm-api-key-rotation-example"
  project_id    = "<project-id>"
  environment   = "<environment-slug>"
  secret_path   = "<secret-path>" # Root folder is /
  connection_id = "<app-connection-id>"

  parameters = {
    name    = "<litellm-api-key-name>"
    team_id = "<litellm-team-id>"       # Optional
    user_id = "<litellm-user-id>"       # Optional
    models  = ["gpt-4o", "gpt-4o-mini"] # Optional, omit to allow all models

    # Optional extra /key/generate options
    additional_options = jsonencode({
      max_budget = 100
      tpm_limit  = 10000
      metadata = {
        owner = "platform-team"
      }
    })
  }

  secrets_mapping = {
    api_key = "LITELLM_API_KEY"
  }
}
