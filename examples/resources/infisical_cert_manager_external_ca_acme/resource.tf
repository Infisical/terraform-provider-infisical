resource "infisical_project" "pki" {
  name        = "PKI Project"
  slug        = "pki-project"
  type        = "cert-manager"
  description = "Project for managing SSL/TLS certificates"
}

# First create an app connection for DNS challenge validation (example with Route53)
# This would be created separately or referenced from another module

resource "infisical_cert_manager_external_ca_acme" "letsencrypt" {
  project_slug = infisical_project.pki.slug

  name   = "letsencrypt-prod"
  status = "active"

  dns_app_connection_id = "your-route53-connection-id"
  dns_provider          = "route53" # Supported values: route53, cloudflare, dns-made-easy
  dns_hosted_zone_id    = "Z123456789ABCDEFGH"

  directory_url = "https://acme-v02.api.letsencrypt.org/directory"
  account_email = "admin@acme.com"

  # External Account Binding (optional, required by some CAs)
  # eab_kid      = "your-eab-key-id"
  # eab_hmac_key = "your-eab-hmac-key"
}