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

variable "keystore_password" {
  type      = string
  sensitive = true
}

resource "infisical_certificate_sync_windows_server" "example" {
  name           = "windows-server-certificate-sync-demo"
  description    = "Demo of Windows Server certificate sync"
  application_id = "<cert-manager-application-id>"
  connection_id  = "<winrm-app-connection-id>"

  destination_config = {
    destination_path = "C:\\certs"
    # host, port and the ssl_* settings pick the target per sync and are only valid with an LDAP connection
  }

  sync_options = {
    certificate_name_schema = "{{commonName}}-{{shortCertificateId}}"
    export_format           = "pkcs12"
    file_access_rules = [
      {
        identity = "CORP\\svc-iis"
        access   = "read"
      },
      {
        identity = "BUILTIN\\Administrators"
        access   = "full-control"
      },
    ]
  }

  export_password = var.keystore_password # Stored in state, use export_password_wo to avoid that

  certificate_filters = {
    certificate_ids = [infisical_cert_manager_certificate.example.id]
  }
}
