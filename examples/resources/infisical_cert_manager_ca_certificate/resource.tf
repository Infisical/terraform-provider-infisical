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

resource "infisical_cert_manager_internal_ca" "root" {
  type          = "root"
  name          = "enterprise-root-ca"
  common_name   = "Enterprise Root Certificate Authority"
  organization  = "Example Corp"
  ou            = "IT Security"
  country       = "US"
  locality      = "San Francisco"
  province      = "California"
  key_algorithm = "RSA_2048"
}

resource "infisical_cert_manager_internal_ca" "issuing" {
  type          = "intermediate"
  name          = "enterprise-issuing-ca"
  common_name   = "Enterprise Issuing Certificate Authority"
  organization  = "Example Corp"
  ou            = "IT Security"
  country       = "US"
  locality      = "San Francisco"
  province      = "California"
  key_algorithm = "RSA_2048"
}

resource "infisical_cert_manager_ca_certificate" "root_cert" {
  ca_id = infisical_cert_manager_internal_ca.root.id

  not_before = "2024-01-01T00:00:00Z"
  not_after  = "2034-01-01T00:00:00Z"

  max_path_length = 1
}

resource "infisical_cert_manager_ca_certificate" "issuing_cert" {
  ca_id        = infisical_cert_manager_internal_ca.issuing.id
  parent_ca_id = infisical_cert_manager_internal_ca.root.id

  not_before = "2024-01-01T00:00:00Z"
  not_after  = "2029-01-01T00:00:00Z"

  max_path_length = 0

  depends_on = [infisical_cert_manager_ca_certificate.root_cert]
}

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
