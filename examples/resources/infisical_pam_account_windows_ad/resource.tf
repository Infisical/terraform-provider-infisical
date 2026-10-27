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

resource "infisical_pam_account_windows_ad" "pamadmin" {
  name        = "corp-pamadmin"
  folder_id   = "<folder-id>"
  template_id = "<windows-ad-template-id>"

  connection_details = {
    domain     = "corp.example.com"
    dc_address = "dc01.corp.example.com"
    hosts      = ["ws01.corp.example.com", "ws02.corp.example.com"]
  }

  credentials = {
    username = "CORP\\pamadmin"
    password = var.pamadmin_password
  }

  self_rotation = true
}

variable "pamadmin_password" {
  type      = string
  sensitive = true
}
