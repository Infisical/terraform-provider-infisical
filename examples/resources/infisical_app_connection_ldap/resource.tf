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
  method      = "simple-bind"
  credentials = {
    provider                = "active-directory"
    url                     = "ldap://ldap.example.com:389"
    dn                      = "cn=admin,dc=example,dc=com"
    password                = "<password>"
    ssl_reject_unauthorized = false
  }
}

# Example with LDAPS (secure LDAP)
resource "infisical_app_connection_ldap" "ldap-demo-secure" {
  name        = "ldap-demo-secure"
  description = "This is a demo LDAP connection with SSL."
  method      = "simple-bind"
  credentials = {
    provider                = "active-directory"
    url                     = "ldaps://ldap.example.com:636"
    dn                      = "cn=admin,dc=example,dc=com"
    password                = "<password>"
    ssl_reject_unauthorized = true
    ssl_certificate         = file("${path.module}/ca.pem")
  }
}
