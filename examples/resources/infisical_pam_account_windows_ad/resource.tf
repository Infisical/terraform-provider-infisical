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

resource "infisical_pam_folder" "windows_team" {
  name = "windows-team"
}

# Windows AD sessions only launch when the template records to S3.
resource "infisical_pam_account_template" "windows_ad" {
  name         = "windows-ad-default"
  account_type = "windows-ad"
  gateway_id   = data.infisical_gateway.main.id

  settings = {
    recording = {
      storage_backend = "aws-s3"
      connection_id   = "<aws-app-connection-id>"
      s3 = {
        bucket     = "acme-pam-recordings"
        region     = "us-east-1"
        key_prefix = "windows/"
      }
    }
  }
}

resource "infisical_pam_account_windows_ad" "pamadmin" {
  name        = "corp-pamadmin"
  folder_id   = infisical_pam_folder.windows_team.id
  template_id = infisical_pam_account_template.windows_ad.id

  connection_details = {
    domain     = "corp.example.com"
    dc_address = "dc01.corp.example.com"
    hosts      = ["ws01.corp.example.com", "ws02.corp.example.com"]
  }

  credentials = {
    username = "CORP\\pamadmin"
    password = var.pamadmin_password
  }

  rotation_account_id = "self"
}

variable "pamadmin_password" {
  type      = string
  sensitive = true
}
