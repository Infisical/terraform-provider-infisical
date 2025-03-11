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
      client_id     = "f1db897a-bac9-4be8-9ce0-274a9d1f1c62"
      client_secret = "2d7048a5f6bde2948ef43ab636bf1c6cce6d1d0bf7aee170a5403642ff7e1d1f"
    }
  }
}

resource "infisical_secret_sync_gcp_secret_manager" "secret_manager_testsss" {
  name          = "gcp-sync-testsaaa"
  description   = "I am a test secret sync"
  project_id    = "207ecdf1-bf06-4a5b-840d-4a52c9c32447"
  environment   = "dev"
  secret_path   = "/"
  connection_id = "e5f707e2-b83d-4b12-9279-f6acfde429cd"

  sync_options = {
    initial_sync_behavior = "overwrite-destination"
  }
  destination_config = {
    project_id = "daniel-test-437412"
  }
}
