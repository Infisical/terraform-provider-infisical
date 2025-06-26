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


resource "infisical_dynamic_secret_kubernetes" "kubernetes" {
  name             = "kubernetes-dynamic-secret-example"
  project_slug     = "your-project-slug"
  environment_slug = "dev"
  path             = "/"
  default_ttl      = "1h"
  max_ttl          = "4h"

  configuration = {
    # This parameter is used if 'auth_method' is set to "gateway"
    # gateway_id = ""

    auth_method = "api"
    api_config = {
      cluster_url   = "https://example.com"
      cluster_token = "<token>"
      enable_ssl    = false
      ca            = ""
    }

    credential_type = "static"
    static_config = {
      service_account_name = "test-account"
      namespace            = "default"
    }

    # This block is used if 'credential_type' is set to "dynamic"
    # dynamic_config = {
    #   allowed_namespaces = "default,namespace2"
    #   role               = "test-role"
    #   role_type          = "role"
    # }

    audiences = []
  }

  username_template = "{{randomUsername}}"
}
