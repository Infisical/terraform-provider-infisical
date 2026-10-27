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

resource "infisical_pam_account_kubernetes" "prod" {
  name        = "prod-cluster-admin"
  folder_id   = "<folder-id>"
  template_id = "<kubernetes-template-id>"

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
