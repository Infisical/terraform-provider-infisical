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

resource "infisical_pam_account_template" "mongodb" {
  name         = "mongodb"
  account_type = "mongodb"
  gateway_id   = data.infisical_gateway.main.id
}

resource "infisical_pam_account_mongodb" "orders" {
  name        = "orders-admin"
  folder_id   = infisical_pam_folder.dbas.id
  template_id = infisical_pam_account_template.mongodb.id

  connection_details = {
    connection_string = "mongodb://mongo-1.internal:27017,mongo-2.internal:27017/?replicaSet=rs0"
    database          = "orders"
    ssl_certificate   = file("mongo-ca.pem")
  }

  credentials = {
    username = "orders_admin"
    password = var.orders_admin_password
  }
}

variable "orders_admin_password" {
  type      = string
  sensitive = true
}
