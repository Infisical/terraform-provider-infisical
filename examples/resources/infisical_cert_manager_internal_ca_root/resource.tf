resource "infisical_project" "pki" {
  name        = "PKI Project"
  slug        = "pki-project"
  type        = "cert-manager"
  description = "Project for managing SSL/TLS certificates"
}

resource "infisical_cert_manager_internal_ca_root" "root" {
  project_slug = infisical_project.pki.slug

  name          = "enterprise-root-ca"
  friendly_name = "Enterprise Root CA"
  common_name   = "Enterprise Root Certificate Authority"
  organization  = "Acme Corp"
  ou            = "IT Security"
  country       = "US"
  province      = "California"
  locality      = "San Francisco"
  key_algorithm = "RSA_2048"
}