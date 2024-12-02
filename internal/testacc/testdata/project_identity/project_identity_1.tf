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

resource "infisical_identity" "test_identity" {
  name   = "test_identity"
  role   = "member"
  org_id = var.org_id
}

resource "infisical_project_identity" "test_identity" {
  project_id  = infisical_project.test_project.id
  identity_id = infisical_identity.test_identity.id
  roles = [
    {
      role_slug = "member"
    }
  ]
}
