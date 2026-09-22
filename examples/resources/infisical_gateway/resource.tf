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

# A gateway record is the thing everything else references. Deploy the gateway itself separately
# with the Helm chart or the CLI and point it at this resource's id, so rebuilding the machine
# never changes the id that app connections and dynamic secrets are pinned to.

# AWS: an EC2 instance authenticates with its instance role on every start.
resource "infisical_gateway" "aws" {
  name = "prod-us-east"

  aws_auth = {
    allowed_account_ids    = ["123456789012"]
    allowed_principal_arns = ["arn:aws:iam::123456789012:role/infisical-gateway"]
  }
}

# GCP: a Compute Engine VM or a GKE pod with workload identity presents a metadata ID token.
# A zone on its own restricts nothing, so at least one service account or project is required.
resource "infisical_gateway" "gcp" {
  name = "prod-gce"

  gcp_auth = {
    type                     = "gce" # or "iam", for hosts outside Compute Engine
    allowed_service_accounts = ["infisical-gateway@my-project.iam.gserviceaccount.com"]
    allowed_projects         = ["my-project"]
    allowed_zones            = ["us-central1-a"]
  }
}

# Kubernetes: Infisical reviews the pod's projected service account token against the cluster's
# API server.
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

# The same, for a cluster whose API server Infisical cannot reach: an already-connected gateway
# in that cluster performs the TokenReview with its own service account, so no host or reviewer
# token is needed. The reviewing gateway has to be a different one, and the first gateway in a
# cluster therefore has to use the api mode above.
resource "infisical_gateway" "gke_second" {
  name = "gke-prod-2"

  kubernetes_auth = {
    token_review_mode             = "gateway"
    reviewer_gateway_id           = infisical_gateway.gke.id
    allowed_namespaces            = ["infisical"]
    allowed_service_account_names = ["infisical-gateway"]
  }
}

# Token: for machines no cloud or cluster can vouch for. The token itself is a separate resource.
resource "infisical_gateway" "datacenter" {
  name = "datacenter-01"

  token_auth = {}
}

# Downstream resources reference the record, not the machine, so an instance rebuild leaves them
# untouched.
output "gateway_id" {
  value = infisical_gateway.aws.id
}
