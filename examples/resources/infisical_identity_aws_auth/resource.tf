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

resource "infisical_identity" "machine-identity-1" {
  name   = "machine-identity-1"
  role   = "admin"
  org_id = "<your-org-id>"
}

resource "infisical_identity_aws_auth" "aws-auth" {
  identity_id            = infisical_identity.machine-identity-1.id
  sts_endpoint           = "https://sts.us-east-1.amazonaws.com/"
  allowed_account_ids    = ["123456789012"]
  allowed_principal_arns = ["arn:aws:iam::123456789012:role/MyRole"]
  access_token_ttl       = 2592000
  access_token_max_ttl   = 2592000

  access_token_trusted_ips = [
    { ip_address = "0.0.0.0/0" }
  ]
}
