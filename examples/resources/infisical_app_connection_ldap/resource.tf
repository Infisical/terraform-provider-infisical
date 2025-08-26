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

resource "infisical_app_connection_ldap" "ldap-demo" {
  name        = "ldap-demo"
  description = "This is a demo LDAP connection."
  method      = "bind-credentials"
  credentials = {
    host          = "ldap.example.com"
    port          = 389
    bind_dn       = "cn=admin,dc=example,dc=com"
    bind_password = "<password>"
    base_dn       = "dc=example,dc=com"
    tls_enabled   = false
  }
}

# Example with TLS enabled
resource "infisical_app_connection_ldap" "ldap-demo-tls" {
  name        = "ldap-demo-tls"
  description = "This is a demo LDAP connection with TLS."
  method      = "bind-credentials"
  credentials = {
    host            = "ldaps.example.com"
    port            = 636
    bind_dn         = "cn=admin,dc=example,dc=com"
    bind_password   = "<password>"
    base_dn         = "dc=example,dc=com"
    tls_enabled     = true
    tls_skip_verify = false
    tls_ca          = file("${path.module}/ca.crt")
  }
}