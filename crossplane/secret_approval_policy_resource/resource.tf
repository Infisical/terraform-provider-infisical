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
  project_id        = "c9a38c17-0cea-4130-9b7a-059f9f1c8fbd"
  name              = "my-secret-approval-policy"
  environment_slugs = ["prod"]
  secret_path       = "/"

  group_approvers = [
    "6629e33a-ab1d-4d0d-958b-63640e7988db",
  ]
  user_approvers = [
    "matheus@infisical.com",
  ]

  group_bypassers = [
    "6629e33a-ab1d-4d0d-958b-63640e7988db",
  ]
  user_bypassers = [
    "admin@infisical.com",
  ]

  required_approvals  = 2
  enforcement_level   = "soft"
  allow_self_approval = false
}
