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

variable "oidc_client_secret" {
  type      = string
  sensitive = true
}

# Users can log in through OIDC only with an email domain that the organization has verified.
resource "infisical_email_domain" "example" {
  domain = "example.com"
}

# Publish verification_record_name and verification_code as a DNS TXT record with your DNS provider,
# and add that record to depends_on here.
resource "infisical_email_domain_verification" "example" {
  email_domain_id = infisical_email_domain.example.id
}

# An identity provider with a discovery endpoint.
resource "infisical_sso_oidc" "example" {
  discovery_url = "https://idp.example.com/.well-known/openid-configuration"
  client_id     = "<oidc-client-id>"

  # Write-only: the secret is not stored in state. Requires Terraform 1.11 or later.
  # Increment client_secret_wo_version when the secret changes.
  client_secret_wo         = var.oidc_client_secret
  client_secret_wo_version = 1

  allowed_email_domains = ["example.com"]
  is_active             = true

  # Changing only the MFA attributes keeps SSO enforcement on.
  mfa_amr_values = ["otp", "hwk"]

  # Configure OIDC after the domain is verified.
  depends_on = [infisical_email_domain_verification.example]
}

# An identity provider without a discovery endpoint.
# resource "infisical_sso_oidc" "custom" {
#   custom = {
#     issuer                 = "https://idp.example.com"
#     authorization_endpoint = "https://idp.example.com/authorize"
#     token_endpoint         = "https://idp.example.com/token"
#     jwks_uri               = "https://idp.example.com/jwks"
#     userinfo_endpoint      = "https://idp.example.com/userinfo"
#   }
#   client_id     = "<oidc-client-id>"
#   client_secret = var.oidc_client_secret
#   is_active     = true
#
#   depends_on = [infisical_email_domain_verification.example]
# }
