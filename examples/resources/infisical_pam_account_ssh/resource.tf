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

resource "infisical_pam_account_ssh" "web01" {
  name        = "web01-root"
  folder_id   = "<folder-id>"
  template_id = "<ssh-template-id>"

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
