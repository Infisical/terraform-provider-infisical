terraform {
  required_providers {
    infisical = {
      # version = <latest version>
      source = "infisical/infisical"
    }
  }
}

provider "infisical" {
  host = "http://localhost:8080" # Only required if using self hosted instance of Infisical, default is https://app.infisical.com
  auth = {
    universal = {
      client_id     = "<machine-identity-client-id>"
      client_secret = "<machine-identity-client-secret>"
    }
  }
}

resource "infisical_identity" "machine-identity-jwks-1" {
  name   = "machine-identity-jwks-1"
  role   = "admin"
  org_id = "a797dc61-9a3a-424f-bc1a-46b39ea8d369"
}

resource "infisical_identity" "machine-identity-static-1" {
  name   = "machine-identity-static-1"
  role   = "admin"
  org_id = "a797dc61-9a3a-424f-bc1a-46b39ea8d369"
}

# JWKS configuration example
resource "infisical_identity_jwt_auth" "jwt-auth-jwks-1" {
  identity_id        = infisical_identity.machine-identity-jwks-1.id
  configuration_type = "jwks"
  jwks_url           = "https://example.com/.well-known/jwks.json"
  jwks_ca_cert       = file("public.pem")
  bound_issuer       = "https://example.com"
  bound_audiences    = ["my-audience"]
  bound_subject      = "my-subject"
}

# Static configuration example
resource "infisical_identity_jwt_auth" "jwt-auth-static-1" {
  identity_id        = infisical_identity.machine-identity-static-1.id
  configuration_type = "static"
  public_keys        = [file("public.pem")]
  bound_issuer       = "https://example.com"
  bound_audiences    = ["my-audience"]
}
