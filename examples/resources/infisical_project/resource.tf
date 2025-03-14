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
      client_id     = "f66716d8-874d-4456-9f59-5b5185e2518c"
      client_secret = "18a880b034e5d367065047972c021212707054bb89ce61d24f42e853ed50b6bb"
    }
  }
}


resource "infisical_project" "te11st" {
  description   = ""
  name          = "aasdaaaasda"
  template_name = "test"
  slug          = "aaasdasda-aaaaboqj"
}

resource "infisical_project_user" "test-user" {
  project_id = infisical_project.te11st.id
  username   = "dani250g@hotmail.com"
  roles = [
    {
      role_slug = "admin"
    }
  ]
}
