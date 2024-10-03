terraform {
  required_providers {
    infisical = {
      # version = <latest version>
      source = "infisical/infisical"
    }
  }
}

provider "infisical" {
  host          = "https://app.infisical.com" # Only required if using self hosted instance of Infisical, default is https://app.infisical.com
  client_id     = "<machine-identity-client-id>"
  client_secret = "<machine-identity-client-secret>"
}

variable "service_account_json" {
  type        = string
  description = "Google Cloud service account JSON key"
}



resource "infisical_integration_gcp_secret_manager" "gcp-integration" {
  project_id           = "your-project-id"
  service_account_json = var.service_account_json
  environment          = "dev"
  secret_path          = "/"
  gcp_project_id       = "gcp-project-id"

}