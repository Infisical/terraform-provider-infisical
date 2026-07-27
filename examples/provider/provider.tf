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
    # organization_slug = "<organization slug to scope the session to sub-org>"
    universal = {
      client_id     = "<machine-identity-client-id>"
      client_secret = "<machine-identity-client-secret>"
    }
  }
}

# Alternatively, select the auth method and supply its credentials via environment variables:
#
# provider "infisical" {
#   auth_method = "universal" # token | universal | oidc | kubernetes | aws_iam
# }
#
# export INFISICAL_AUTH_METHOD="universal"  # or set auth_method as above
# export INFISICAL_UNIVERSAL_AUTH_CLIENT_ID="<machine-identity-client-id>"
# export INFISICAL_UNIVERSAL_AUTH_CLIENT_SECRET="<machine-identity-client-secret>"
