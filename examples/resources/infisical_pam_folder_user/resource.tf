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

resource "infisical_pam_folder" "linux_team" {
  name = "linux-team"
}

resource "infisical_pam_user" "alice" {
  email = "alice@example.com"
  role  = "member"
}

resource "infisical_pam_folder_user" "alice_linux" {
  folder_id = infisical_pam_folder.linux_team.id
  email     = infisical_pam_user.alice.email # also makes Terraform add the user to PAM first
  role      = "operator"
}
