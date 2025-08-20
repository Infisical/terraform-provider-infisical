terraform {
  required_providers {
    infisical = {
      source = "infisical/infisical"
    }
  }
}

provider "infisical" {
  host = "https://app.infisical.com" # Only required if using self hosted instance of Infisical

  auth = {
    universal = {
      client_id     = var.infisical_client_id
      client_secret = var.infisical_client_secret
    }
  }
}

variable "infisical_client_id" {
  type = string
}

variable "infisical_client_secret" {
  type      = string
  sensitive = true
}

variable "project_id" {
  type        = string
  description = "The ID of your Infisical project"
}

variable "ldap_connection_id" {
  type        = string
  description = "The ID of your LDAP app connection"
}

resource "infisical_secret_rotation_ldap_password" "example" {
  name          = "ldap-password-rotation"
  description   = "Rotation for LDAP user passwords"
  project_id    = var.project_id
  connection_id = var.ldap_connection_id
  environment   = "prod"
  secret_path   = "/ldap"

  auto_rotation_enabled = true
  rotation_interval     = 30 # days

  rotate_at_utc = {
    hours   = 2
    minutes = 0
  }

  parameters = {
    dn = "cn=service-user,ou=users,dc=example,dc=com"

    password_requirements = {
      length = 24

      required = {
        digits    = 4
        lowercase = 4
        uppercase = 4
        symbols   = 2
      }

      allowed_symbols = "!@#$%^&*()_-+=[]{}|:;<>?,./"
    }

    rotation_method = "modify" # or "reset" depending on your LDAP setup
  }

  secrets_mapping = {
    dn       = "LDAP_USER_DN"
    password = "LDAP_USER_PASSWORD"
  }
}