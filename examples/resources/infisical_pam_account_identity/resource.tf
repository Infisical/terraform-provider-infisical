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

data "infisical_gateway" "main" {
  name = "prod-gateway"
}

resource "infisical_pam_folder" "linux_team" {
  name = "linux-team"
}

resource "infisical_pam_account_template" "ssh" {
  name         = "ssh-default"
  account_type = "ssh"
  gateway_id   = data.infisical_gateway.main.id
}

resource "infisical_pam_account_ssh" "web01" {
  name        = "web01-root"
  folder_id   = infisical_pam_folder.linux_team.id
  template_id = infisical_pam_account_template.ssh.id

  connection_details = {
    host = "10.0.1.10"
    port = 22
  }

  credentials = {
    auth_method = "public-key"
    username    = "root"
    private_key = var.web01_private_key
  }
}

variable "web01_private_key" {
  type      = string
  sensitive = true
}

resource "infisical_pam_identity" "ci" {
  identity_id = "<identity-id>"
  role        = "member"
}

resource "infisical_pam_account_identity" "ci_web01" {
  account_id  = infisical_pam_account_ssh.web01.id
  identity_id = infisical_pam_identity.ci.identity_id # also makes Terraform add the identity to PAM first
  role        = "connector"
}
