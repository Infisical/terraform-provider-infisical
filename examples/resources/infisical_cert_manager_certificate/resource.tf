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

# Request a certificate
resource "infisical_cert_manager_certificate" "my_cert" {
  profile_id          = "<internal-ca-profile-id>"
  common_name         = "api.example.com"
  alt_names           = ["api.example.com", "api-internal.example.com"]
  organization        = "Example Corp"
  country             = "US"
  key_algorithm       = "RSA_2048"
  signature_algorithm = "RSA-SHA256"
  ttl                 = "90d"
  timeout_seconds     = 300
}


# Request a certificate using a CSR file
resource "infisical_cert_manager_certificate" "csr_based_cert" {
  profile_id      = "<profile-id>"
  csr_path        = "./my-certificate.csr"
  timeout_seconds = 300
}

# Outputs
output "cert_details" {
  value = {
    id            = infisical_cert_manager_certificate.my_cert.id
    status        = infisical_cert_manager_certificate.my_cert.status
    serial_number = infisical_cert_manager_certificate.my_cert.serial_number
    not_before    = infisical_cert_manager_certificate.my_cert.not_before
    not_after     = infisical_cert_manager_certificate.my_cert.not_after
  }
}

output "cert_pem" {
  value     = infisical_cert_manager_certificate.my_cert.certificate
  sensitive = true
}

output "cert_chain" {
  value     = infisical_cert_manager_certificate.my_cert.certificate_chain
  sensitive = true
}

output "cert_private_key" {
  value     = infisical_cert_manager_certificate.my_cert.private_key
  sensitive = true
}
