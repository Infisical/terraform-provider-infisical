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

resource "infisical_secret_folder" "folder1" {
  name             = var.folder_name
  environment_slug = "dev"
  project_id       = infisical_project.test_project.id
  folder_path      = "/"
}

resource "infisical_secret_folder" "folder2" {
  name             = var.folder_name
  environment_slug = "dev"
  project_id       = infisical_project.test_project.id
  folder_path      = "/${infisical_secret_folder.folder1.name}"
}

resource "infisical_secret_folder" "folder3" {
  name             = var.folder_name
  environment_slug = "dev"
  project_id       = infisical_project.test_project.id
  folder_path      = "/${infisical_secret_folder.folder1.name}/${infisical_secret_folder.folder2.name}"
}

variable "folder_name" {
  type = string
}
