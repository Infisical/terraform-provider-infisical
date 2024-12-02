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

resource "infisical_secret_tag" "tag" {
  slug       = var.secret_tag_slug
  name       = var.secret_tag_slug
  color      = "#fff"
  project_id = infisical_project.test_project.id
}

variable "secret_tag_slug" {
  type = string
}

resource "infisical_secret" "test_secret" {
  name         = var.secret_name
  value        = var.secret_value
  env_slug     = var.secret_env_slug
  workspace_id = infisical_project.test_project.id
  folder_path  = var.secret_path
  tag_ids      = [infisical_secret_tag.tag.id]
}

variable "secret_name" {
  type = string
}

variable "secret_value" {
  type      = string
  sensitive = true
}

variable "secret_env_slug" {
  type = string
}

variable "secret_path" {
  type = string
}
