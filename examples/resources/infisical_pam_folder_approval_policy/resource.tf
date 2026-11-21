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

# Accounts using this template need an approved request, and the bypassers can break glass on them.
resource "infisical_pam_account_template" "prod_ssh" {
  name         = "prod-ssh"
  account_type = "ssh"

  policies = {
    requires_approval = true
    allow_break_glass = true
  }
}

resource "infisical_pam_user" "lead" {
  email = "lead@example.com"
  role  = "member"
}

resource "infisical_pam_user" "oncall" {
  email = "oncall@example.com"
  role  = "member"
}

resource "infisical_pam_group" "dbas" {
  group_id = "<group-id>"
  role     = "member"
}

resource "infisical_pam_folder_user" "lead" {
  folder_id = infisical_pam_folder.prod.id
  email     = infisical_pam_user.lead.email # also makes Terraform add the user to PAM first
  role      = "admin"
}

resource "infisical_pam_folder_user" "oncall" {
  folder_id = infisical_pam_folder.prod.id
  email     = infisical_pam_user.oncall.email
  role      = "operator"
}

resource "infisical_pam_folder_group" "dbas" {
  folder_id = infisical_pam_folder.prod.id
  group_id  = infisical_pam_group.dbas.group_id
  role      = "operator"
}

resource "infisical_pam_folder_approval_policy" "prod" {
  folder_id = infisical_pam_folder.prod.id

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
