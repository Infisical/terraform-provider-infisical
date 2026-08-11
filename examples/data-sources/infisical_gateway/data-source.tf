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

data "infisical_gateway" "example" {
  name = "<gateway-name>"
}

output "gateway-id" {
  value = data.infisical_gateway.example.id
}

# The resolved ID can be passed anywhere a gateway is referenced, such as a dynamic secret
resource "infisical_dynamic_secret_sql_database" "sql-database" {
  name             = "postgres-dynamic-secret"
  project_slug     = "<project-slug>"
  environment_slug = "prod"
  path             = "/"

  configuration = {
    client               = "postgres"
    host                 = "postgres.internal"
    port                 = "5432"
    database             = "infisical"
    username             = "infisical"
    password             = "infisical"
    gateway_id           = data.infisical_gateway.example.id
    creation_statement   = <<-EOT
      CREATE USER "{{username}}" WITH ENCRYPTED PASSWORD '{{password}}' VALID UNTIL '{{expiration}}';
      GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO "{{username}}";
    EOT
    revocation_statement = <<-EOT
      REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM "{{username}}";
      DROP ROLE "{{username}}";
    EOT
  }
}
