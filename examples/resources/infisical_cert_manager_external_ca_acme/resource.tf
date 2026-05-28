terraform {
  required_providers {
    infisical = {
      source = "infisical/infisical"
    }
  }
}

provider "infisical" {
  host          = "https://app.infisical.com"
  client_id     = var.client_id
  client_secret = var.client_secret
}

resource "infisical_cert_manager_external_ca_acme" "letsencrypt" {
  name   = "letsencrypt-prod"
  status = "active"

  dns_app_connection_id = "your-route53-connection-id"
  dns_provider          = "route53" # Supported values: route53, cloudflare, dns-made-easy
  dns_hosted_zone_id    = "Z123456789ABCDEFGH"

  directory_url = "https://acme-v02.api.letsencrypt.org/directory"
  account_email = "admin@acme.com"
}
