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

variable "role_slug" {
  type = string
}

resource "infisical_project_role" "test" {
  project_slug = infisical_project.test_project.slug
  name         = var.role_slug
  description  = "A test role"
  slug         = var.role_slug
  permissions = [
    {
      subject = "secrets"
      action  = "read"
      conditions = {
        secret_path = "/"
        environment = "dev"
      }
    }
  ]
}
