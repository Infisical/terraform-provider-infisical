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


data "infisical_secret_validation_rules" "rules" {
  project_id = "<project-id>"

  # Omit to return every rule in the project, of any type
  type = "static-secrets"
}

output "secret-validation-rules" {
  value = data.infisical_secret_validation_rules.rules.rules
}
