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

# The record only. Deploy the gateway itself with the Helm chart or CLI, pointed at its id.

# AWS: the instance authenticates with its instance role on every start.
resource "infisical_gateway" "aws" {
  name = "prod-us-east"

  aws_auth = {
    allowed_account_ids    = ["123456789012"]
    allowed_principal_arns = ["arn:aws:iam::123456789012:role/infisical-gateway"]
  }
}

# GCP: a metadata ID token from Compute Engine or GKE workload identity.
resource "infisical_gateway" "gcp" {
  name = "prod-gce"

  gcp_auth = {
    type                     = "gce" # or "iam", for hosts outside Compute Engine
    allowed_service_accounts = ["infisical-gateway@my-project.iam.gserviceaccount.com"]
    allowed_projects         = ["my-project"]
    allowed_zones            = ["us-central1-a"]
  }
}

# Kubernetes: Infisical reviews the pod's service account token against the API server.
resource "infisical_gateway" "gke" {
  name = "gke-prod"

  kubernetes_auth = {
    kubernetes_host               = "https://10.0.0.1"
    ca_certificate                = file("${path.module}/cluster-ca.crt")
    token_reviewer_jwt            = var.token_reviewer_jwt
    allowed_namespaces            = ["infisical"]
    allowed_service_account_names = ["infisical-gateway"]
  }
}

# For a cluster Infisical cannot reach: another connected gateway in it does the TokenReview.
resource "infisical_gateway" "gke_second" {
  name = "gke-prod-2"

  kubernetes_auth = {
    token_review_mode             = "gateway"
    reviewer_gateway_id           = infisical_gateway.gke.id
    allowed_namespaces            = ["infisical"]
    allowed_service_account_names = ["infisical-gateway"]
  }
}

# Token: for machines no cloud or cluster can vouch for.
resource "infisical_gateway" "datacenter" {
  name = "datacenter-01"

  token_auth = {}
}

output "gateway_id" {
  value = infisical_gateway.aws.id
}
