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

resource "infisical_access_approval_policy" "prod-policy" {
  project_id        = "b394bb99-3ee4-4fc6-8adf-dae14c59ce9a"
  name              = "my-approval-policy"
  environment_slugs = ["prod"]
  secret_path       = "/"
  approvers = [
    {
      type = "group"
      id   = "7c13f73b-c09b-4752-aea6-9b691ba3eb45"
      step = 1
    },
    {
      type = "group"
      id   = "83d5cf3b-3580-4aaf-872e-564a8ccaaf86"
      step = 2
    },
    {
      type     = "user"
      username = "admin@infisical.com"
      step     = 1
  }]
  bypassers = [
    {
      type = "group"
      id   = "7c13f73b-c09b-4752-aea6-9b691ba3eb45"
    }
  ]
  required_approvals  = 1
  enforcement_level   = "soft"
  allow_self_approval = true

  approvals_required = [
    {
      number_of_approvals = 2
      step_number         = 1
    },
    {
      number_of_approvals = 1
      step_number         = 2
    }
  ]

  max_time_period         = "24h"
  request_expiration_time = "72h"
}
