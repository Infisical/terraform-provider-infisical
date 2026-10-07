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

# Infisical calls the Kubernetes API server directly
resource "infisical_identity_kubernetes_auth_template" "api" {
  name                      = "prod-cluster"
  kubernetes_host           = "https://<your-kubernetes-host>:6443"
  kubernetes_ca_certificate = file("<path-to-kubernetes-ca-certificate>")
  token_reviewer_jwt        = "ey<example>"
  allowed_audience          = "infisical"
}

# The TokenReview runs through a gateway deployed in the cluster
resource "infisical_identity_kubernetes_auth_template" "gateway" {
  name                = "private-cluster"
  token_reviewer_mode = "gateway"
  gateway_id          = "<your-gateway-id>"
}

# Identities take the connection settings from the template and keep their own allowlists
resource "infisical_identity" "machine-identity-demo" {
  name   = "machine-identity-demo"
  role   = "admin"
  org_id = "<your-org-id>"
}

resource "infisical_identity_kubernetes_auth" "kubernetes-auth-demo" {
  identity_id = infisical_identity.machine-identity-demo.id
  template_id = infisical_identity_kubernetes_auth_template.api.id

  allowed_namespaces            = ["infisical-ns"]
  allowed_service_account_names = ["infisical-sa"]
}
