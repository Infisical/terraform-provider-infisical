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

resource "infisical_dynamic_secret_aws_iam" "aws-iam" {
  name             = "aws-iam-dynamic-secret-example"
  project_slug     = "your-project-slug"
  environment_slug = "dev"
  path             = "/"
  default_ttl      = "2h"
  max_ttl          = "4h"

  configuration = {
    method = "access_key"

    # This block is used if 'method' is set to "access_key"
    access_key_config = {
      access_key        = "YOUR_AWS_ACCESS_KEY_ID"
      secret_access_key = "YOUR_AWS_SECRET_ACCESS_KEY"
    }

    # This block is used if 'method' is set to "assume_role"
    # assume_role_config = {
    #   role_arn = "arn:aws:iam::123456789012:role/YourAssumeRole"
    # }

    region = "us-east-1"

    aws_path                       = "/"
    permission_boundary_policy_arn = "arn:aws:iam::123456789012:policy/YourBoundaryPolicy"
    policy_document                = <<-EOT
    {
      "Version": "2012-10-17",
      "Statement": [
        {
          "Effect": "Allow",
          "Action": "s3:ListBucket",
          "Resource": "*"
        }
      ]
    }
    EOT
    user_groups                    = "group-a,group-b"
    policy_arns                    = "arn:aws:iam::aws:policy/ReadOnlyAccess,arn:aws:iam::123456789012:policy/SpecificPolicy"
  }

  username_template = "{{randomUsername}}"
}
