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

# Infisical accepts enforce_sso = true only after someone has logged in through the active SSO configuration.
# Apply with enforce_sso = false first, log in once through your identity provider, then set it to true.
resource "infisical_sso_settings" "example" {
  enforce_sso              = true
  admin_sso_bypass_enabled = true
  sso_mfa_trust            = "signal"
}
