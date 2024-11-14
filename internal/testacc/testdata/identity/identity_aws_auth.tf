resource "infisical_project" "test_project" {
  name = var.project_name
  slug = var.project_slug
}

variable "project_name" {
  type = string
}

variable "project_slug" {
  type = string
}

variable "org_id" {
  type = string
}

resource "infisical_identity" "aws-auth" {
  name   = "aws-auth"
  role   = "member"
  org_id = var.org_id
}

resource "infisical_identity_aws_auth" "aws-auth" {
  identity_id                 = infisical_identity.aws-auth.id
  access_token_ttl            = 2592000
  access_token_max_ttl        = 2592000 * 2
  access_token_num_uses_limit = 3
  allowed_principal_arns      = ["arn:aws:iam::123456789012:user/MyUserName"]
  allowed_account_ids         = ["123456789012", "123456789013"]
}
