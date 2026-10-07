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

resource "infisical_certificate_sync_linux_server" "example" {
  name           = "linux-server-certificate-sync-demo"
  description    = "Demo of Linux Server certificate sync"
  application_id = "<cert-manager-application-id>"
  connection_id  = "<ssh-app-connection-id>"

  destination_config = {
    destination_path = "/etc/ssl/infisical"
    # host, port and ssh_host_keys pick the target per sync and are only valid with an LDAP connection
  }

  sync_options = {
    certificate_name_schema = "{{commonName}}-{{shortCertificateId}}"
    can_remove_certificates = true
    export_format           = "jks"
    keystore_alias          = "tomcat"
    include_truststore      = true
    file_mode               = "0644"
    private_key_file_mode   = "0600"
    owner                   = "tomcat"
    group                   = "tomcat"
    post_sync_command       = "systemctl reload tomcat"
  }

  export_password_wo         = var.keystore_password
  export_password_wo_version = 1 # Increment to send a new export_password_wo

  certificate_filters = {
    certificate_ids = [infisical_cert_manager_certificate.example.id]
  }
}
