terraform {
  required_providers {
    infisical = {
      # version = <latest version>
      source = "infisical/infisical"
    }
  }
}

provider "infisical" {
  host = "https://app.infisical.com" # Only required if using self hosted instance of Infisical, default is https://app.infisical.com
  auth = {
    universal = {
      client_id     = "<machine-identity-client-id>"
      client_secret = "<machine-identity-client-secret>"
    }
  }
}

resource "infisical_project" "example" {
  name = "example"
  slug = "example"
}

resource "infisical_project_identity" "test-identity" {
  project_id  = infisical_project.example.id
  identity_id = "<identity id>"
  roles = [
    {
      role_slug = "admin"
    }
  ]
}

# When the machine identity that runs Terraform is the one that created the
# project, Infisical automatically adds it as an admin member. In that case,
# a plain resource block would fail with a conflict error. Setting
# adopt_existing = true tells the provider to adopt that pre-existing
# membership and update its roles to match this configuration instead.
resource "infisical_project_identity" "creator-identity" {
  project_id     = infisical_project.example.id
  identity_id    = "<creator-identity-id>"
  adopt_existing = true
  roles = [
    {
      role_slug = "admin"
    }
  ]
}
