terraform {
  required_providers {
    infisical = {
      # version = <latest version>
      source = "infisical/infisical"
    }
    aws = {
      source = "hashicorp/aws"
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

data "infisical_instance_deployment_aws_metadata" "this" {
  # account_id is null when the instance has no AWS identity. Stop with a clear message
  # instead of building a trust policy from it (postconditions need Terraform 1.2 or later).
  lifecycle {
    postcondition {
      condition     = self.account_id != null
      error_message = "This Infisical instance has no AWS identity. Run it on AWS with an instance role, or set the AWS App Connection keys (INF_APP_CONNECTION_AWS_ACCESS_KEY_ID and INF_APP_CONNECTION_AWS_SECRET_ACCESS_KEY)."
    }
  }
}

# Trust policy for an IAM role that an Infisical AWS App Connection assumes.
data "aws_iam_policy_document" "infisical_trust" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "AWS"
      identifiers = ["arn:aws:iam::${data.infisical_instance_deployment_aws_metadata.this.account_id}:root"]
    }

    condition {
      test     = "StringEquals"
      variable = "sts:ExternalId"
      values   = ["<your-infisical-project-id>"]
    }
  }
}

output "infisical_aws_account_id" {
  value = data.infisical_instance_deployment_aws_metadata.this.account_id
}
