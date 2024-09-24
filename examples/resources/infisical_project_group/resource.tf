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
  group_id   = "<>"
  roles = [
    {
      role_slug                   = "admin",
      is_temporary                = true,
      temporary_access_start_time = "2024-09-19T12:43:13Z",
      temporary_range             = "1y"
    },
    {
      role_slug = "my-custom-role",
    },
  ]
}
