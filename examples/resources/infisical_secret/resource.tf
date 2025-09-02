terraform {
  required_providers {
    infisical = {
      # version = <latest version>
      source = "hashicorp.com/edu/infisical"
    }
  }
}

provider "infisical" {
  host = "https://app.infisical.com" # Only required if using self hosted instance of Infisical, default is https://app.infisical.com
  auth = {
    universal = {
      client_id     = "b84dd7c6-a773-4870-9da3-e49192844c6c"
      client_secret = "8e40851114c90312f970bdca9453fe5a0c404c5017a6025977654b62eabaf322"
    }
  }
}

resource "infisical_secret" "mongo_secret" {
  name         = "MONGO_DB-new"
  value        = "<some-key>2"
  env_slug     = "dev"
  workspace_id = "5156a345-e460-416b-84fc-b14b426b1cb3"
  folder_path  = "/"
}
