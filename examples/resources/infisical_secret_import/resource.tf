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

# prod:/app reads the secrets of dev:/shared in the same project.
resource "infisical_secret_import" "same-project" {
  # Destination: where the secrets are imported into.
  project_id       = "<your-infisical-project-id>"
  environment_slug = "prod"
  folder_path      = "/app"

  # Source: where the secrets are imported from.
  import_environment_slug = "dev"
  import_folder_path      = "/shared"
  is_replication          = false
}

# prod:/app reads the secrets of prod:/database in another project.
# Needs cross-project secret sharing and a project grant in the source project.
resource "infisical_secret_import" "cross-project" {
  # Destination: where the secrets are imported into.
  project_id       = "<your-infisical-project-id>"
  environment_slug = "prod"
  folder_path      = "/app"

  # Source: where the secrets are imported from.
  import_project_id       = "<source-infisical-project-id>"
  import_environment_slug = "prod"
  import_folder_path      = "/database"
  is_replication          = false
}
