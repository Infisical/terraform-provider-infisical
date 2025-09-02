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

resource "infisical_dynamic_secret_mongo_atlas" "mongo-atlas" {
  name             = "mongo-atlas-dynamic-secret"
  project_slug     = "your-project-slug"
  environment_slug = "dev"
  path             = "/"
  default_ttl      = "1h"
  max_ttl          = "24h"

  configuration = {
    admin_public_key  = "<your-mongo-atlas-public-key>"
    admin_private_key = "<your-mongo-atlas-private-key>"
    group_id          = "<your-mongo-atlas-project-id"

    roles = [
      {
        database_name = "test"
        role_name     = "readWrite"
      },
      {
        database_name = "admin"
        role_name     = "read"
      }
    ]

    # Required - specify clusters or data lakes the user can access
    scopes = [
      {
        name = "myCluster1"
        type = "CLUSTER"
      },
      {
        name = "myCluster2"
        type = "CLUSTER"
      }
    ]
  }

  username_template = "{{randomUsername}}"
}
