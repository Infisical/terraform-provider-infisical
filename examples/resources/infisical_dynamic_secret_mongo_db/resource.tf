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

resource "infisical_dynamic_secret_mongo_db" "mongo-db" {
  name             = "mongo-db-dynamic-secret-example"
  project_slug     = "your-project-slug"
  environment_slug = "dev"
  path             = "/"
  default_ttl      = "1h"
  max_ttl          = "24h"

  configuration = {
    host     = "your-host"
    port     = 27017
    username = "your-username"
    password = "your-password"
    database = "default"
    roles    = ["readWrite"]
  }

  username_template = "{{randomUsername}}"
}
