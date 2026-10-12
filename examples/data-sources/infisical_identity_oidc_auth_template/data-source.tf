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

data "infisical_identity_oidc_auth_template" "by-name" {
  name = "<template-name>"
}

data "infisical_identity_oidc_auth_template" "by-id" {
  id = "<template-id>"
}

resource "infisical_identity_oidc_auth" "oidc-auth-demo" {
  identity_id   = "<identity-id>"
  template_id   = data.infisical_identity_oidc_auth_template.by-name.id
  bound_subject = "<expected-subject>"
}
