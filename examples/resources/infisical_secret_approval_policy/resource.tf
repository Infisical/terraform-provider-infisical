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

resource "infisical_secret_approval_policy" "prod-policy" {
  project_id        = infisical_project.example.id
  name              = "my-prod-policy"
  environment_slugs = ["prod"]
  secret_path       = "/"
  approvers = [
    {
      type = "group"
      id   = "52c70c28-9504-4b88-b5af-ca2495dd277d"
    },
    {
      type     = "user"
      username = "name@infisical.com"
  }]
  required_approvals = 1
  enforcement_level  = "hard"
}
