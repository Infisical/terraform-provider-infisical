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
    universal_auth = {
      client_id     = "<machine-identity-client-id>"
      client_secret = "<machine-identity-client-secret>"
    }
  }
}


resource "infisical_access_approval_policy" "prod-policy" {
  project_id        = "5156a345-e460-416b-84fc-b14b426b1cb3"
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
      type     = "user"
      username = "admin@infisical.com"
      step     = 2
    },
    {
      type = "group"
      id   = "83d5cf3b-3580-4aaf-872e-564a8ccaaf86"
      step = 2
    },
  ]

  required_approvals = 1
  enforcement_level  = "soft"
  allow_self_approval = false

  approvals_required = [
    {
      number_of_approvals = 2
      step_number         = 1
    },
     {
      number_of_approvals = 2
      step_number         = 2
    }
  ]

  max_time_period           = "24h"
  request_expiration_time   = "72h"
}
