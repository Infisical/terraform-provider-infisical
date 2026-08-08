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
#
# Authenticate via Universal Auth:
# export INFISICAL_AUTH_METHOD="universal"
# export INFISICAL_UNIVERSAL_AUTH_CLIENT_ID="<machine-identity-client-id>"
# export INFISICAL_UNIVERSAL_AUTH_CLIENT_SECRET="<machine-identity-client-secret>"
#
# Authenticate via Token Auth:
# export INFISICAL_AUTH_METHOD="token"
# export INFISICAL_TOKEN="<machine-identity-token>"
#
# Authenticate via OIDC Auth:
# export INFISICAL_AUTH_METHOD="oidc"
# export INFISICAL_MACHINE_IDENTITY_ID="<machine-identity-id>"
# export INFISICAL_AUTH_JWT="<oidc-jwt>"
#
# Authenticate via Kubernetes Auth:
# export INFISICAL_AUTH_METHOD="kubernetes"
# export INFISICAL_MACHINE_IDENTITY_ID="<machine-identity-id>"
# export INFISICAL_KUBERNETES_SERVICE_ACCOUNT_TOKEN="<service-account-token>"
# # Or use the token file path (defaults to /var/run/secrets/kubernetes.io/serviceaccount/token):
# # export INFISICAL_KUBERNETES_SERVICE_ACCOUNT_TOKEN_PATH="<path-to-service-account-token>"
#
# Authenticate via AWS IAM Auth:
# export INFISICAL_AUTH_METHOD="aws_iam"
# export INFISICAL_MACHINE_IDENTITY_ID="<machine-identity-id>"
