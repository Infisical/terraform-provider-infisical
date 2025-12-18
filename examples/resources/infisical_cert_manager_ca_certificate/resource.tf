resource "infisical_project" "pki" {
  name        = "PKI Project"
  slug        = "pki-project"
  description = "Project for PKI certificate management"
  type        = "cert-manager"
}

resource "infisical_cert_manager_internal_ca_root" "root" {
  project_slug = infisical_project.pki.slug

  name          = "enterprise-root-ca"
  common_name   = "Enterprise Root Certificate Authority"
  organization  = "Example Corp"
  ou            = "IT Security"
  country       = "US"
  locality      = "San Francisco"
  province      = "California"
  key_algorithm = "RSA_2048"
}

resource "infisical_cert_manager_internal_ca_intermediate" "issuing" {
  project_slug = infisical_project.pki.slug
  parent_ca_id = infisical_cert_manager_internal_ca_root.root.id

  name          = "enterprise-issuing-ca"
  common_name   = "Enterprise Issuing Certificate Authority"
  organization  = "Example Corp"
  ou            = "IT Security"
  country       = "US"
  locality      = "San Francisco"
  province      = "California"
  key_algorithm = "RSA_2048"
}

# Generate certificate for the root CA
resource "infisical_cert_manager_ca_certificate" "root_cert" {
  ca_id = infisical_cert_manager_internal_ca_root.root.id

  not_before = "2024-01-01T00:00:00Z"
  not_after  = "2034-01-01T00:00:00Z"

  # Root CA can issue intermediate CAs (path length = 1)
  max_path_length = 1
}

# Generate certificate for the intermediate CA
resource "infisical_cert_manager_ca_certificate" "issuing_cert" {
  ca_id = infisical_cert_manager_internal_ca_intermediate.issuing.id

  not_before = "2024-01-01T00:00:00Z"
  not_after  = "2029-01-01T00:00:00Z"

  # Intermediate CA cannot issue further intermediates (path length = 0)
  max_path_length = 0

  # Ensure root certificate is generated first
  depends_on = [infisical_cert_manager_ca_certificate.root_cert]
}

# Example with shorter validity period for testing/development
resource "infisical_cert_manager_ca_certificate" "dev_cert" {
  ca_id = infisical_cert_manager_internal_ca_root.root.id

  not_before = "2024-01-01T00:00:00Z"
  not_after  = "2025-01-01T00:00:00Z" # 1 year validity

  # No path length restriction (-1 = unlimited)
  max_path_length = -1
}

# Output the certificate information
output "root_ca_certificate" {
  description = "The root CA certificate"
  value       = infisical_cert_manager_ca_certificate.root_cert.certificate
  sensitive   = true
}

output "root_ca_serial_number" {
  description = "The serial number of the root CA certificate"
  value       = infisical_cert_manager_ca_certificate.root_cert.serial_number
}

output "issuing_ca_certificate" {
  description = "The issuing CA certificate"
  value       = infisical_cert_manager_ca_certificate.issuing_cert.certificate
  sensitive   = true
}

output "issuing_ca_chain" {
  description = "The issuing CA certificate chain"
  value       = infisical_cert_manager_ca_certificate.issuing_cert.certificate_chain
  sensitive   = true
}
