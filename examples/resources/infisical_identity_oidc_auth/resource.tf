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
  client_id     = "<>"
  client_secret = "<>"
}

resource "infisical_project" "example" {
  name = "example"
  slug = "example"
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
