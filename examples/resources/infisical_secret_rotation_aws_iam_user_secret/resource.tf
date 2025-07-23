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

resource "infisical_secret_rotation_aws_iam_user_secret" "aws-iam-user-secret" {
  name          = "aws-iam-user-secret-rotation-example"
  project_id    = "<project-id>"
  environment   = "<environment-slug>"
  secret_path   = "<secret-path>" # Root folder is /
  connection_id = "<app-connection-id>"

  parameters = {
    user_name = "<aws-iam-user-name>"
    region    = "<aws-region>" # e.g. us-east-1
  }

  secrets_mapping = {
    access_key_id     = "AWS_ACCESS_KEY_ID"
    secret_access_key = "AWS_SECRET_ACCESS_KEY"
  }
}
