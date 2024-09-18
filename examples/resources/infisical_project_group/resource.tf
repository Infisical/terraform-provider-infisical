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

resource "infisical_project" "example" {
  name = "example"
  slug = "example"
}

resource "infisical_project_group" "group" {
  project_id = infisical_project.example.id
  group_slug = "my-group"
  roles = [
    {
      role_slug                   = "admin",
      is_temporary                = true,
      temporary_access_start_time = "<>",
      temporary_range             = "<>"
    },
    {
      role_slug = "my-custom-role",
    },
  ]
}
