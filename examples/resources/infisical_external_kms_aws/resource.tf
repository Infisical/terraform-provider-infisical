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

resource "infisical_external_kms_aws" "external-kms-aws-assume-role" {
  name        = "aws-assume-role-external-kms"
  description = "AWS External KMS using assume-role type"
  configuration = {
    aws_region     = "us-east-1"
    aws_kms_key_id = "<aws-kms-key-id>"
    type           = "assume-role"
    credential = {
      role_arn         = "<assume-role-arn>"
      role_external_id = "<role-external-id>"
    }
  }
}

resource "infisical_external_kms_aws" "external-kms-aws-access-key" {
  name        = "aws-access-key-external-kms"
  description = "AWS External KMS using access-key type"
  configuration = {
    aws_region     = "us-east-1"
    aws_kms_key_id = "<aws-kms-key-id>"
    type           = "access-key"
    credential = {
      access_key_id     = "<access-key-id>"
      secret_access_key = "<secret-access-key>"
    }
  }
}

