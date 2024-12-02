
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

locals {
  environment_slugs = ["dev", "prod"]
}

resource "infisical_secret_folder" "folder1" {
  for_each = toset(local.environment_slugs)

  name             = "nested"
  environment_slug = each.value
  project_id       = infisical_project.test_project.id
  folder_path      = "/"
}

resource "infisical_secret_import" "import-3" {
  environment_slug        = "dev"
  import_environment_slug = "prod"
  is_replication          = false
  project_id              = infisical_project.test_project.id
  folder_path             = "/"
  import_folder_path      = "/nested"
}

resource "infisical_secret_import" "nested-import-1" {
  environment_slug        = "dev"
  import_environment_slug = "prod"
  is_replication          = false
  project_id              = infisical_project.test_project.id
  folder_path             = "/nested"
  import_folder_path      = "/nested"
}
