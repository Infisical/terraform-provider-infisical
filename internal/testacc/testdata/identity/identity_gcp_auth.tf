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

resource "infisical_identity" "gcp-auth" {
  name   = "gcp-auth"
  role   = "member"
  org_id = var.org_id
}

resource "infisical_identity_gcp_auth" "gcp-auth" {
  identity_id = infisical_identity.gcp-auth.id
  type        = "gce"
}
