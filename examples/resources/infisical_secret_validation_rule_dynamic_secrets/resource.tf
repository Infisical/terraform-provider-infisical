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

resource "infisical_secret_validation_rule_dynamic_secrets" "dynamic-secrets" {
  name        = "dynamic-secrets-validation-rule-example"
  project_id  = "<project-id>"
  secret_path = "/**" # Supports glob patterns such as /apps/**

  constraints = {
    # A lease is only constrained when its provider is listed here
    providers = ["sql-database"]

    # These replace any password requirements configured on the dynamic secret itself
    password_constraints = {
      min_length = 24
      max_length = 64
    }
  }
}
