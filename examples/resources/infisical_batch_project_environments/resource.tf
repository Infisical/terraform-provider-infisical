terraform {
  required_providers {
    infisical = {
      # version = <latest version>
      source = "hashicorp.com/edu/infisical"
    }
  }
}

provider "infisical" {
  host          = "http://localhost:8080" # Only required if using self hosted instance of Infisical, default is https://app.infisical.com
  client_id     = "b7a7e3ed-9aa0-4ff2-90f9-c7094dd60536"
  client_secret = "253325e9ad5856512df27cdf9301ef1696a855a139bf30cac791fafe067a067c"
}


resource "infisical_batch_project_environment" "pre-prod" {
  project_id = "bdca54d1-b802-4f45-b94c-c81b311e7761"
  environments = [
  

    {
      name = "pre-prodXD1233"
      slug = "pre-prod-slug"
    }
  ]
}
