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

resource "infisical_pam_folder" "cloud" {
  name = "cloud"
}

resource "infisical_pam_account_template" "aws" {
  name         = "aws-console"
  account_type = "aws-iam"

  policies = {
    max_session_duration = 3600
  }
}

resource "infisical_pam_account_aws_iam" "billing" {
  name        = "billing-read-only"
  folder_id   = infisical_pam_folder.cloud.id
  template_id = infisical_pam_account_template.aws.id

  connection_details = {
    # The role's trust policy must allow Infisical, with your organization ID as the external ID.
    role_arn = "arn:aws:iam::123456789012:role/infisical-pam-billing"
  }
}
