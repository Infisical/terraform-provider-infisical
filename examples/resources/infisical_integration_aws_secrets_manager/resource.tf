terraform {
  required_providers {
    infisical = {
      # version = <latest version>
      source = "infisical/infisical"
    }
  }
}

provider "infisical" {
  host          = "https://app.infisical.com" # Only required if using self hosted instance of Infisical, default is https://app.infisical.com
  client_id     = "<machine-identity-client-id>"
  client_secret = "<machine-identity-client-secret>"
}

resource "infisical_integration_aws_secrets_manager" "secrets-manager-integration" {
  project_id  = "<project-id>"
  aws_region  = "<aws-region>" // example, us-east-2
  environment = "<env-slug>"   // example, dev

  secret_path = "<infisical-secrets-path>" // example, /folder, or /

  secrets_manager_path = "/example/secrets" # Only required if mapping_behavior is one-to-one
  mapping_behavior     = "one-to-one"       # Optional, default is many-to-one

  # AWS Authentication
  access_key_id     = "<aws-access-key-id>"
  secret_access_key = "<aws-secret-access-key>"
  # OR
  assume_role_arn = "arn:aws:iam::<aws-account-id>:role/<role-name>"

  options = {
    secret_prefix = "<optional-prefix>"
    aws_tags = [
      {
        key   = "key",
        value = "value"
      },
    ]
  }
}