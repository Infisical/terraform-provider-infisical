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

resource "infisical_secret_rotation_gcp_service_account_key" "gcp-service-account-key" {
  name          = "gcp-service-account-key-rotation-example"
  project_id    = "<project-id>"
  environment   = "<environment-slug>"
  secret_path   = "<secret-path>" # Root folder is /
  connection_id = "<app-connection-id>"

  parameters = {
    service_account_email = "my-app@my-project.iam.gserviceaccount.com"
  }

  secrets_mapping = {
    service_account_key = "GCP_SERVICE_ACCOUNT_KEY"
  }
}
