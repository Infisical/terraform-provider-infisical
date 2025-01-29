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

resource "infisical_app_connection_gcp" "app-connection-gcp" {
  name                  = "gcp-app-connection"
  method                = "service-account-impersonation"
  service_account_email = "service-account-df92581a-0fe9@my-duplicate-project.iam.gserviceaccount.com"
  description           = "I am a default description"
}
