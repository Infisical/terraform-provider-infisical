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

data "infisical_gateway" "main" {
  name = "prod-gateway"
}

resource "infisical_pam_folder" "devops_team" {
  name = "devops-team"
}

resource "infisical_pam_account_template" "kubernetes" {
  name         = "kubernetes-admin"
  account_type = "kubernetes"
  gateway_id   = data.infisical_gateway.main.id
}

resource "infisical_pam_account_kubernetes" "prod" {
  name        = "prod-cluster-admin"
  folder_id   = infisical_pam_folder.devops_team.id
  template_id = infisical_pam_account_template.kubernetes.id

  connection_details = {
    url             = "https://k8s.prod.internal:6443"
    ssl_certificate = file("prod-ca.pem")
  }

  credentials = {
    auth_method          = "gateway-kubernetes-auth"
    namespace            = "infisical"
    service_account_name = "pam-admin"
  }
}
