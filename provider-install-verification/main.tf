terraform {
  required_providers {
    infisical = {
      source = "hashicorp.com/edu/infisical"
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
