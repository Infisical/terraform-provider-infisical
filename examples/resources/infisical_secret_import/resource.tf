terraform {
  required_providers {
    infisical = {
      # version = <latest version>
      source = "infisical/infisical"
    }
  }
}

provider "infisical" {
  host = "https://app.infisical.com" # Only required if using self hosted instance of Infisical, default is https://app.infisical.com
  auth = {
    universal = {
      client_id     = "<machine-identity-client-id>"
      client_secret = "<machine-identity-client-secret>"
    }
  }
}

# Import the secrets of /shared in dev into /app in prod of the same project.
resource "infisical_secret_import" "same-project" {
  project_id              = "<your-infisical-project-id>"
  environment_slug        = "prod"
  folder_path             = "/app"
  import_environment_slug = "dev"
  import_folder_path      = "/shared"
  is_replication          = false
}

# Import the secrets of /database in prod of another project.
# The organization must have cross-project secret sharing turned on, and the
# source project needs a project grant that shares prod:/database with this project.
resource "infisical_secret_import" "cross-project" {
  project_id              = "<your-infisical-project-id>"
  environment_slug        = "prod"
  folder_path             = "/app"
  import_project_id       = "<source-infisical-project-id>"
  import_environment_slug = "prod"
  import_folder_path      = "/database"
  is_replication          = false
}
