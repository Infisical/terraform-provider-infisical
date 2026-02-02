resource "infisical_project" "pki" {
  name        = "PKI Project"
  slug        = "pki-project"
  type        = "cert-manager"
  description = "Project for managing SSL/TLS certificates"
}

# Root CA example
resource "infisical_cert_manager_internal_ca" "root" {
  project_slug = infisical_project.pki.slug

  type          = "root"
  name          = "enterprise-root-ca"
  common_name   = "Enterprise Root Certificate Authority"
  organization  = "Acme Corp"
  ou            = "IT Security"
  country       = "US"
  province      = "California"
  locality      = "San Francisco"
  key_algorithm = "RSA_2048"
}

# Intermediate CA example
resource "infisical_cert_manager_internal_ca" "issuing" {
  project_slug = infisical_project.pki.slug

  type          = "intermediate"
  name          = "enterprise-issuing-ca"
  common_name   = "Enterprise Issuing Certificate Authority"
  organization  = "Acme Corp"
  ou            = "IT Security"
  country       = "US"
  province      = "California"
  locality      = "San Francisco"
  key_algorithm = "RSA_2048"
}
