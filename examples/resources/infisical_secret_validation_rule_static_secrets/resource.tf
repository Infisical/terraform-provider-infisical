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

resource "infisical_secret_validation_rule_static_secrets" "static-secrets" {
  name        = "static-secrets-validation-rule-example"
  project_id  = "<project-id>"
  environment = "<environment-slug>" # Omit to enforce the rule in every environment of the project
  secret_path = "/**"                # Supports glob patterns such as /apps/**

  constraints = {
    key_constraints = {
      regex_pattern   = "^[A-Z][A-Z0-9_]*$"
      required_prefix = "APP_"
    }

    value_constraints = {
      min_length = 16
      max_length = 128

      reuse_prevention = {
        previous_versions = 10
      }
    }
  }
}
