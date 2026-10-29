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

resource "infisical_pam_folder" "dbas" {
  name = "dbas"
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
  }
}

resource "infisical_pam_account_postgres" "admin" {
  name        = "orders-admin"
  folder_id   = infisical_pam_folder.dbas.id
  template_id = infisical_pam_account_template.postgres.id

  connection_details = {
    host     = "orders.db.internal"
    database = "orders"
  }

  credentials = {
    username = "orders_admin"
    password = var.orders_admin_password
  }

  self_rotation = true
}

resource "infisical_pam_account_postgres" "app" {
  name        = "orders-app"
  folder_id   = infisical_pam_folder.dbas.id
  template_id = infisical_pam_account_template.postgres.id

  connection_details = {
    host     = "orders.db.internal"
    database = "orders"
  }

  credentials = {
    username = "orders_app"
    password = var.orders_app_password
  }

  # The admin account rotates this account's password.
  rotation_account_id = infisical_pam_account_postgres.admin.id
}

resource "infisical_pam_account_postgres" "rds" {
  name        = "reporting-iam"
  folder_id   = infisical_pam_folder.dbas.id
  template_id = infisical_pam_account_template.postgres.id

  connection_details = {
    host     = "reporting.abc123.us-east-1.rds.amazonaws.com"
    database = "reporting"
  }

  credentials = {
    auth_method = "aws-iam"
    username    = "reporting"
    aws_region  = "us-east-1"
    role_arn    = "arn:aws:iam::123456789012:role/infisical-pam-rds"
  }
}

variable "orders_admin_password" {
  type      = string
  sensitive = true
}

variable "orders_app_password" {
  type      = string
  sensitive = true
}
