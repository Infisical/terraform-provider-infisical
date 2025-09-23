terraform {
  required_providers {
    infisical = {
      # version = <latest version>
      source = "hashicorp.com/edu/infisical"
    }
  }
}

provider "infisical" {
  host = "http://localhost:8080" # Only required if using self hosted instance of Infisical, default is https://app.infisical.com
  auth = {
    universal = {
      client_id     = "0627787b-63bb-45d2-90c7-7e47cbb68dc6"
      client_secret = "91e1e75064d8d9982b5ae141f9eb0dbe7c200e065fce3c2783fb2d9e82e47efb"
    }
  }
}

resource "infisical_secret" "write-only-secret" {
  name             = "WRITE-ONLY-SECRET"
  env_slug         = "prod-bk03"
  value_wo         = "test-valuetestddsssssaaadddssaa"
  value_wo_version = 1
  workspace_id     = "cc7c320b-b0c1-422f-831a-a51d536dc3c2"
  folder_path      = "/"
}

resource "infisical_secret" "normal-secret" {
  name         = "NORMAL-SECRET"
  env_slug     = "prod-bk03"
  value        = "test-valuetestddsssssaaadddssaa"
  workspace_id = "cc7c320b-b0c1-422f-831a-a51d536dc3c2"
  folder_path  = "/"
}


