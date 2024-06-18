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

data "infisical_secret_tag" "terraform" {
  slug       = "terraform"
  project_id = "PROJECT_ID"
}

output "secret-tag" {
  value = data.infisical_secret_tag.terraform
}

