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

resource "infisical_project_group" "group" {
  project_id = infisical_project.example.id

  # Either group_id or group_name is required.
  group_id   = "<>"
  group_name = "<>"
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
