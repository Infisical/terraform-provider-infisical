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


resource "infisical_integration_aws_parameter_store" "parameter-store-integration" {
  project_id  = "<project-id>"
  environment = "<env-slug>" // example, dev

  secret_path          = "<infisical-secrets-path>" // example, /folder, or /
  parameter_store_path = "/example/secrets"

  aws_region = "<aws-region>" // example, us-east-2

  # AWS Authentication
  access_key_id     = "<aws-access-key-id>"
  secret_access_key = "<aws-secret-access-key>"
  # OR
  assume_role_arn = "arn:aws:iam::<aws-account-id>:role/<role-name>"


  // Optional
  options = {
    should_disable_delete = true // Optional, default is false
    aws_tags = [                 // Optional
      {
        key   = "key",
        value = "value"
      },
    ]
  }
}