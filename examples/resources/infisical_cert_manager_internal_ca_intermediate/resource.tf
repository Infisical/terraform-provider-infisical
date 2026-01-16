resource "infisical_project" "pki" {
  name        = "PKI Project"
  slug        = "pki-project"
  type        = "cert-manager"
  description = "Project for managing SSL/TLS certificates"
}

resource "infisical_cert_manager_internal_ca_root" "root" {
  project_slug = infisical_project.pki.slug

  name          = "enterprise-root-ca"
  common_name   = "Enterprise Root Certificate Authority"
  organization  = "Acme Corp"
  ou            = "IT Security"
  country       = "US"
  province      = "California"
  locality      = "San Francisco"
  key_algorithm = "RSA_2048"
}

resource "infisical_cert_manager_internal_ca_intermediate" "issuing" {
  project_slug = infisical_project.pki.slug

  name          = "enterprise-issuing-ca"
  common_name   = "Enterprise Issuing Certificate Authority"
  organization  = "Acme Corp"
  ou            = "IT Security"
  country       = "US"
  province      = "California"
  locality      = "San Francisco"
  key_algorithm = "RSA_2048"
}