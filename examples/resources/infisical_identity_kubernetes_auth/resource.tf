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


# Using normal API as token reviewer mode
resource "infisical_identity" "machine-identity-demo-1" {
  name   = "machine-identity-demo-1"
  role   = "admin"
  org_id = "<your-org-id>"
}

resource "infisical_identity_kubernetes_auth" "kubernetes-auth-demo-1" {
  identity_id               = infisical_identity.machine-identity-1.id
  kubernetes_host           = "http://<your-kubernetes-host>"
  token_reviewer_jwt        = "ey<example>"
  kubernetes_ca_certificate = "<your-kubernetes-ca-certificate>"

  allowed_namespaces            = ["infisical-ns"]
  allowed_service_account_names = ["infisical-sa", "infisical-sa-2"]
  access_token_ttl              = 2592000
  access_token_max_ttl          = 2592000

  token_reviewer_mode = "api"
}


# Using gateway as token reviewer mode
resource "infisical_identity" "machine-identity-demo-2" {
  name   = "machine-identity-demo-2"
  role   = "admin"
  org_id = "<your-org-id>"
}

# When using gateway as reviewer, you do not need to specify the kubernetes host, token reviewer JWT, or CA certificate as they're all provided by the gateway.
resource "infisical_identity_kubernetes_auth" "kubernetes-auth-demo-2" {
  identity_id = infisical_identity.machine-identity-2.id

  allowed_namespaces            = ["infisical-ns"]
  allowed_service_account_names = ["infisical-sa", "infisical-sa-2"]
  access_token_ttl              = 2592000
  access_token_max_ttl          = 2592000

  token_reviewer_mode = "gateway"
  gateway_id          = "<your-gateway-id>"
}

resource "infisical_identity_kubernetes_auth" "import" {}
