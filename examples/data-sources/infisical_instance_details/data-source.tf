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

data "infisical_instance_details" "this" {}

# Trust policy for an IAM role that an Infisical AWS App Connection assumes.
data "aws_iam_policy_document" "infisical_trust" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "AWS"
      identifiers = ["arn:aws:iam::${data.infisical_instance_details.this.aws_account_id}:root"]
    }

    condition {
      test     = "StringEquals"
      variable = "sts:ExternalId"
      values   = ["<your-infisical-project-id>"]
    }
  }
}

output "infisical_aws_account_id" {
  value = data.infisical_instance_details.this.aws_account_id
}
