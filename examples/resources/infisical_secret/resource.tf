terraform {
  required_providers {
    infisical = {
      # version = <latest version>
      source = "infisical/infisical"
    }
  }
}

provider "infisical" {
  host          = "http://localhost:8080" # Only required if using self hosted instance of Infisical, default is https://app.infisical.com
  client_id     = "8c1dcb47-e351-4898-b23f-377eb9a6fc1b"
  client_secret = "2d40c95991774ad60a8b64ef68859a79a00df322bc78da002976a12fbafcecf2"
}

resource "infisical_secret" "test_secret" {
  name         = "HELLO2"
  value        = "world"
  env_slug     = "dev"
  workspace_id = "1e5341db-401a-4044-997c-7f229246a178"
  folder_path  = "/"
  secret_reminder = {
    note        = "Rotate this secret using X API"
    repeat_days = 30
  }
}
