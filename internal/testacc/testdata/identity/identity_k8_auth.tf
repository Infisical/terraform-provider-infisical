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

resource "infisical_identity" "k8-auth" {
  name   = "k8-auth"
  role   = "member"
  org_id = var.org_id
}

resource "infisical_identity_kubernetes_auth" "k8-auth" {
  identity_id        = infisical_identity.k8-auth.id
  kubernetes_host    = "http://example.com"
  token_reviewer_jwt = "ey<example>"
  allowed_namespaces = ["namespace-a", "namespace-b"]
}
