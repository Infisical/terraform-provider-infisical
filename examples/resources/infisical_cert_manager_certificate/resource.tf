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

resource "infisical_cert_manager_application" "platform" {
  name        = "platform"
  description = "Certificates issued for the platform team"
}

resource "infisical_cert_manager_certificate" "api_direct" {
  profile_id     = "<profile-id>"
  application_id = infisical_cert_manager_application.platform.id

  common_name         = "api.example.com"
  alt_names           = ["api.example.com", "api-internal.example.com"]
  organization        = "Example Corp"
  country             = "US"
  key_algorithm       = "RSA_2048"
  signature_algorithm = "RSA-SHA256"
  key_usages          = ["digital_signature", "key_encipherment"]
  extended_key_usages = ["server_auth"]
  ttl                 = "90d"
  timeout_seconds     = 300
}

resource "infisical_cert_manager_certificate" "api_csr_file" {
  profile_id     = "<profile-id>"
  application_id = infisical_cert_manager_application.platform.id

  csr             = file("./my-certificate.csr")
  ttl             = "90d"
  timeout_seconds = 300
}

resource "infisical_cert_manager_certificate" "api_csr_inline" {
  profile_id = "<profile-id>"

  csr = <<-CSR
-----BEGIN CERTIFICATE REQUEST-----
MIICljCCAX4CAQAwGDEWMBQGA1UEAwwNaW5maXNpY2FsLmNvbTCCASIwDQYJKoZI
hvcNAQEBBQADggEPADCCAQoCggEBAL7c6thewe78xVXyBQ0ZgmSjlkYCGeJgxot/
QqB1+z3mlMr/tViy8UFBThO69i532ZgWyuJ24YWVeF67WBQPYXnkXlUMJKYGM6UA
ka6dOjCeSJHNXkxbLHI3yljvXP/Kn6w+WczeSuJXYNH5Uet9TtTEuXIqTgWjnAhD
TAWqtyHPbtw3jVWy7xkSc/JuqVGha05snjPPmF9lkdoztG3gosN0TnwTGaQ382sr
giUBkkfdBK0eGmTwJlG9xLZMm3hDyAFz2/iw6GR57uvp/h9RDZVZcuisUQf2T8NZ
CCVPTSGpfYxbQ4KviDghL2/GDY9NVudHSyeCCA0zOPZW9tg7hAECAwEAAaA5MDcG
CSqGSIb3DQEJDjEqMCgwJgYDVR0RBB8wHYINaW5maXNpY2FsLmNvbYIMaW5maXNp
Y2FsLmVzMA0GCSqGSIb3DQEBCwUAA4IBAQBZy+AYPWeZVs+ZPP/9Zj0cl7BchwZV
phEPIezIdKqDyLcyjCf168rbTEqch9gz5CvyIPL3kBohicI/k+/RYPnRKsTdiYNE
XrpeaarHqlvpGzjsrQUv6iJgrDGZXMVJn+op3cDChNPet9RJ1utH96S6W6Ent4QU
90XNi6fBSja8wThfj0AAl51OycHwfNg5/CtygT0eM16/bZl0knJ884Bf35LNE731
Awp8H6ELyXOX1tRKNZRPMKr2Nw/qn6QK611R9aSA+maa25YZa8K0cvSVAJQOfdei
0A7YVKj492nqnN/xS5kzIidZuaCBLocLo5j615xh/79YgMZjrGo3wvnk
-----END CERTIFICATE REQUEST-----
CSR

  ttl             = "90d"
  timeout_seconds = 300
}

output "cert_details" {
  value = {
    id            = infisical_cert_manager_certificate.api_direct.id
    status        = infisical_cert_manager_certificate.api_direct.status
    serial_number = infisical_cert_manager_certificate.api_direct.serial_number
    not_before    = infisical_cert_manager_certificate.api_direct.not_before
    not_after     = infisical_cert_manager_certificate.api_direct.not_after
  }
}

output "cert_pem" {
  value = infisical_cert_manager_certificate.api_direct.certificate
}

output "cert_chain" {
  value = infisical_cert_manager_certificate.api_direct.certificate_chain
}

output "cert_private_key" {
  value     = infisical_cert_manager_certificate.api_direct.private_key
  sensitive = true
}
