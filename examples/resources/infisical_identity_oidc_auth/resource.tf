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

resource "infisical_identity" "machine-identity-1" {
  name   = "machine-identity-1"
  role   = "admin"
  org_id = "<>"
}

resource "infisical_identity_oidc_auth" "oidc-auth" {
  identity_id        = infisical_identity.machine-identity-1.id
  oidc_discovery_url = "<>"
  bound_issuer       = "<>"
  bound_audiences    = ["sample-audience"]
  bound_subject      = "<>"
}

# Using an auth template for the identity provider settings. oidc_discovery_url, bound_issuer,
# bound_audiences and oidc_ca_certificate come from the template, and bound_subject or
# bound_claims must still restrict which workloads can authenticate.
resource "infisical_identity" "machine-identity-2" {
  name   = "machine-identity-2"
  role   = "admin"
  org_id = "<>"
}

resource "infisical_identity_oidc_auth" "oidc-auth-from-template" {
  identity_id   = infisical_identity.machine-identity-2.id
  template_id   = "<your-oidc-auth-template-id>"
  bound_subject = "<>"
}
