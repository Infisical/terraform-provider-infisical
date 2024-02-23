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
  client_id =    "3f6135db-f237-421d-af66-a8f4e80d443b"
  client_secret = "d1a9238d9fe9476e545ba92c25ece3866178e468d3b0b8f263af64026ac835bf"
}

resource "infisical_project" "a-new-project" {
  name              = "new name123"
  slug              = "a-new-project-slug"
  organization_id   = "180870b7-f464-4740-8ffe-9d11c9245ea7"
}


