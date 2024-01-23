terraform {
  required_providers {
    infisical = {
      source = "infisical/infisical"
    }
  }
}

provider "infisical" {
  host          = "https://app.infisical.com"
  service_token = "<>"
}

data "infisical_secrets" "edu" {}

output "secrets" {
  value     = data.infisical_secrets.edu.secrets.maidul
  sensitive = false
}
