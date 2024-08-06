terraform {
  required_providers {
    infisical = {
      # version = <latest version>
      source = "infisical/infisical"
    }
  }
}

provider "infisical" {
  host          = "https://app.infisical.com" # Only required if using self hosted instance of Infisical, default is https://app.infisical.com
  client_id     = "<>"
  client_secret = "<>"
}

resource "infisical_secret_import" "custom-import" {
  environment_slug        = "<ENV_SLUG>"
  import_environment_slug = "<ENV_SLUG>"
  is_replication          = false
  project_id              = "<PROJECT-ID>"
  folder_path             = "<FOLDER_PATH>"
  import_folder_path      = "<IMPORT_FOLDER_PATH>"
}
