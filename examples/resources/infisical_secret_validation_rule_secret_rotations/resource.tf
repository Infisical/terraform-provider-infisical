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

resource "infisical_secret_validation_rule_secret_rotations" "secret-rotations" {
  name        = "secret-rotations-validation-rule-example"
  project_id  = "<project-id>"
  secret_path = "/**" # Supports glob patterns such as /apps/**

  constraints = {
    # A rotation is only constrained when its provider is listed here
    providers = ["postgres-credentials", "ldap-password"]

    # These replace any password requirements configured on the rotation itself
    password_constraints = {
      min_length = 24
    }
  }
}
