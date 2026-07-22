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

resource "infisical_project" "example" {
  name = "example"
  slug = "example"
}

resource "infisical_access_approval_policy" "prod-policy" {
  project_id        = infisical_project.example.id
  name              = "my-approval-policy"
  environment_slugs = ["prod"]
  secret_path       = "/"
  approvers = [
    {
      type = "group"
      id   = "7c13f73b-c09b-4752-aea6-9b691ba3eb45"
    },
    {
      type     = "user"
      username = "admin@infisical.com"
  }]
  bypassers = [
    {
      type = "group"
      id   = "7c13f73b-c09b-4752-aea6-9b691ba3eb45"
    },
    {
      type = "user"
      id   = "admin@infisical.com"
    },
  ]
  required_approvals  = 1
  enforcement_level   = "soft"
  allow_self_approval = true

  max_time_period         = "24h"
  request_expiration_time = "72h"
}
