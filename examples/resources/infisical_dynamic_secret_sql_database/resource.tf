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

resource "infisical_dynamic_secret_sql_database" "sql-database" {
  name             = "postgres-dynamic-secret"
  project_slug     = "project-7-new-c7-vv"
  environment_slug = "prod"
  path             = "/"
  default_ttl      = "2h"
  max_ttl          = "4h"

  configuration = {
    client             = "postgres"
    host               = "host.docker.internal"
    port               = "5431"
    database           = "infisical"
    username           = "infisical"
    password           = "infisical"
    creation_statement = <<-EOT
      CREATE USER "{{username}}" WITH ENCRYPTED PASSWORD '{{password}}' VALID UNTIL '{{expiration}}';
      GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO "{{username}}";
    EOT

    revocation_statement = <<-EOT
      REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM "{{username}}";
      DROP ROLE "{{username}}";
    EOT

    renew_statement = <<-EOT
      ALTER ROLE "{{username}}" VALID UNTIL "{{expiration}}";
    EOT

    password_requirements = {
      length = 32
      required = {
        digits    = 3
        lowercase = 2
        symbols   = 2
        uppercase = 2
      }
      allowed_symbols = "!@#$%^&*()_+-=[]{}|;:,.<>?`~"
    }

  }

  username_template = "{{randomUsername}}"
}
