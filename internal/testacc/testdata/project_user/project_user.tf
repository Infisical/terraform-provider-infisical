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

variable "email" {
  type = string
}


resource "infisical_project_user" "test_user" {
  project_id = infisical_project.test_project.id
  username   = var.email
  roles = [
    {
      role_slug = "admin"
    }
  ]
}
