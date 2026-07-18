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


resource "infisical_secret_approval_policy" "prod-policy" {
  project_id        = "5156a345-e460-416b-84fc-b14b426b1cb3"
  name              = "my-secret-approval-policy"
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
    },
  ]

  bypassers = [
    {
      type     = "user"
      username = "admin@infisical.com"
    },
    {
      type = "group"
      id   = "6629e33a-dc7c-4d0d-918b-62640e6988dc"
    },
  ]

  required_approvals  = 1
  enforcement_level   = "soft"
  allow_self_approval = false
}
