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
    public_key  = "YOUR_MONGO_ATLAS_PUBLIC_KEY"
    private_key = "YOUR_MONGO_ATLAS_PRIVATE_KEY"
    group_id    = "YOUR_MONGO_ATLAS_PROJECT_ID"
    roles       = "readWrite@myDatabase,read@admin"
    scopes      = "myCluster1,myCluster2" # Optional - restrict to specific clusters
  }

  username_template = "{{randomUsername}}"
}