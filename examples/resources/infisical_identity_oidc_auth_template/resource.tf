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

resource "infisical_identity_oidc_auth_template" "github-actions" {
  name               = "github-actions"
  oidc_discovery_url = "https://token.actions.githubusercontent.com"
  bound_issuer       = "https://token.actions.githubusercontent.com"
  bound_audiences    = ["https://github.com/<your-org>"]
}

# Identities take the identity provider settings from the template and bind their own subject or claims
resource "infisical_identity" "machine-identity-demo" {
  name   = "machine-identity-demo"
  role   = "admin"
  org_id = "<your-org-id>"
}

resource "infisical_identity_oidc_auth" "oidc-auth-demo" {
  identity_id   = infisical_identity.machine-identity-demo.id
  template_id   = infisical_identity_oidc_auth_template.github-actions.id
  bound_subject = "repo:<your-org>/<your-repo>:ref:refs/heads/main"
}
