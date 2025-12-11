terraform {
  required_providers {
    infisical = {
      # version = <latest version>
      source = "infisical/infisical"
    }
  }
}

provider "infisical" {
  host = "https://app.infisical.com" # Only required if using self hosted instance of Infisical
  auth = {
    universal = {
      client_id     = "<machine-identity-client-id>"
      client_secret = "<machine-identity-client-secret>"
    }
  }
}

resource "infisical_secret_rotation_ldap_password" "example" {
  name          = "ldap-password-rotation"
  description   = "Rotation for LDAP user passwords"
  project_id    = "<project-id>"
  environment   = "<environment-slug>"
  secret_path   = "<secret-path>" # Root folder is /
  connection_id = "<app-connection-id>"

  auto_rotation_enabled = true
  rotation_interval     = 30 # days

  rotate_at_utc = {
    hours   = 2
    minutes = 0
  }

  parameters = {
    dn = "CN=John,OU=Users,DC=example,DC=com"

    password_requirements = {
      length = 48

      required = {
        digits    = 1
        lowercase = 1
        uppercase = 1
        symbols   = 0
      }

      allowed_symbols = "-_.~!*"
    }

    rotation_method = "connection-principal" # or "target-principal" depending on your LDAP setup
  }

  secrets_mapping = {
    dn       = "LDAP_DN"
    password = "LDAP_PASSWORD"
  }

  # Required when parameters.rotation_method is "target-principal"
  # temporary_parameters = {
  #   password = "<temporary-password-for-target>"
  # }
}
