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

resource "infisical_pam_account_template" "ssh" {
  name         = "ssh-default"
  description  = "Standard SSH access"
  account_type = "ssh"
  gateway_id   = data.infisical_gateway.main.id

  policies = {
    requires_approval    = true
    allow_break_glass    = true
    require_reason       = true
    max_session_duration = 3600
    command_blocking     = ["^rm -rf /", "^shutdown"]
  }

  settings = {
    recording = {
      storage_backend = "aws-s3"
      connection_id   = "<aws-app-connection-id>"
      s3 = {
        bucket     = "acme-pam-recordings"
        region     = "us-east-1"
        key_prefix = "ssh/"
      }
    }

    session_log_masking = {
      patterns           = ["password=\\S+"]
      built_in_detection = true
    }
  }
}

resource "infisical_pam_account_template" "postgres" {
  name         = "postgres-rotated"
  account_type = "postgres"
  gateway_id   = data.infisical_gateway.main.id

  settings = {
    rotation = {
      enabled          = true
      interval_seconds = 604800
    }

    password_requirements = {
      length = 32
      required = {
        digits    = 2
        lowercase = 2
        uppercase = 2
      }
    }
  }
}
