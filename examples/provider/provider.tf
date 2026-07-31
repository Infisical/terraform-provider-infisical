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

# Authenticate via environment variables instead of the auth block:
#
# provider "infisical" {}
#
# export INFISICAL_AUTH_METHOD="universal"  # token | universal | oidc | kubernetes | aws_iam
# export INFISICAL_UNIVERSAL_AUTH_CLIENT_ID="<machine-identity-client-id>"
# export INFISICAL_UNIVERSAL_AUTH_CLIENT_SECRET="<machine-identity-client-secret>"
#
# Other methods (set INFISICAL_AUTH_METHOD accordingly):
#   token:      INFISICAL_TOKEN
#   oidc:       INFISICAL_MACHINE_IDENTITY_ID (+ JWT in INFISICAL_AUTH_JWT by default)
#   kubernetes: INFISICAL_MACHINE_IDENTITY_ID (+ service account token env/path)
#   aws_iam:    INFISICAL_MACHINE_IDENTITY_ID
