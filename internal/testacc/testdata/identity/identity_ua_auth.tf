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

resource "infisical_identity" "universal-auth" {
  name   = "universal-auth"
  role   = "member"
  org_id = var.org_id
}

resource "infisical_identity_universal_auth" "ua" {
  identity_id                 = infisical_identity.universal-auth.id
  access_token_ttl            = 2592000
  access_token_max_ttl        = 2592000 * 2
  access_token_num_uses_limit = 3
}

resource "infisical_identity_universal_auth_client_secret" "client-secret" {
  identity_id = infisical_identity.universal-auth.id

  depends_on = [infisical_identity_universal_auth.ua]
}

output "client_secret" {
  sensitive = true
  value     = infisical_identity_universal_auth_client_secret.client-secret.client_secret
}
