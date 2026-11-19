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

resource "infisical_pam_folder" "prod" {
  name = "prod"
}

resource "infisical_pam_folder_user" "lead" {
  folder_id = infisical_pam_folder.prod.id
  email     = "lead@example.com"
  role      = "admin"
}

resource "infisical_pam_folder_user" "oncall" {
  folder_id = infisical_pam_folder.prod.id
  email     = "oncall@example.com"
  role      = "operator"
}

resource "infisical_pam_folder_group" "dbas" {
  folder_id = infisical_pam_folder.prod.id
  group_id  = "<your-group-id>"
  role      = "operator"
}

resource "infisical_pam_folder_approval_policy" "prod" {
  folder_id = infisical_pam_folder.prod.id
  name      = "prod-access"

  # Approvers must be folder members, and referencing the memberships adds them first.
  approvers = [
    { type = "user", id = infisical_pam_folder_user.lead.user_id },
    { type = "group", id = infisical_pam_folder_group.dbas.group_id },
  ]

  # Can grant themselves access on accounts whose template sets allow_break_glass.
  bypassers = [
    { type = "user", id = infisical_pam_folder_user.oncall.user_id },
  ]

  constraints = {
    access_duration = {
      max = "1d"
    }
  }
}
