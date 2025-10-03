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
  project_id         = "5156a345-e460-416b-84fc-b14b426b1cb3"
  name               = "my-approval-policy"
  environment_slugs  = ["prod"]
  secret_path        = "/"
  approvers          = "[{\"type\":\"group\",\"id\":\"60782603-18bd-4f83-a312-6a9c501f4914\"},{\"type\":\"user\",\"username\":\"vlad@infisical.com\"}]"
  required_approvals = 1
  enforcement_level  = "soft"
}
