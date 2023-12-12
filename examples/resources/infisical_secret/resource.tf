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
  service_token = "<>"
}

resource "infisical_secret_imports" "mongo_import" {
  env_slug    = "dev"
  folder_path = "/apps/api"

  import_secrets {
    env_slug = "dev"
    folder_path = "/db"
  }

  import_secrets {
    env_slug = "dev"
    folder_path = "/mail-services"
  }
}
