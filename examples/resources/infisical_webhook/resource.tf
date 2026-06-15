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

# Notify an external endpoint (e.g. an AWS Lambda) when secrets change
resource "infisical_webhook" "lambda_trigger" {
  project_id         = "<your-project-id>"
  environment        = "prod"
  secret_path        = "/"
  webhook_url        = "https://example.com/infisical-webhook"
  webhook_secret_key = "<signing-key>" # Optional: used to sign the payload so the receiver can verify it
  events_filter      = ["secrets.modified"]
}
