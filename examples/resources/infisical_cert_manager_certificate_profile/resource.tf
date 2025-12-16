resource "infisical_project" "pki" {
  name        = "PKI Project"
  slug        = "pki-project"
  description = "Project for PKI certificate management"
  type        = "cert-manager"
}

resource "infisical_cert_manager_internal_ca_root" "root" {
  project_slug = infisical_project.pki.slug

  name         = "enterprise-root-ca"
  common_name  = "Enterprise Root Certificate Authority"
  organization = "Example Corp"
  ou           = "IT Security"
  country      = "US"
  locality     = "San Francisco"
  province     = "California"
}

resource "infisical_cert_manager_internal_ca_intermediate" "issuing" {
  project_slug = infisical_project.pki.slug
  parent_ca_id = infisical_cert_manager_internal_ca_root.root.id

  name         = "enterprise-issuing-ca"
  common_name  = "Enterprise Issuing Certificate Authority"
  organization = "Example Corp"
  ou           = "IT Security"
  country      = "US"
  locality     = "San Francisco"
  province     = "California"
}

resource "infisical_cert_manager_certificate_template" "web_server" {
  project_slug = infisical_project.pki.slug

  name        = "Web Server Template"
  description = "Template for web server certificates"

  subject {
    type     = "common_name"
    allowed  = ["*.example.com", "*.internal.example.com"]
    required = ["*.example.com"]
  }

  subject {
    type     = "organization"
    required = ["Example Corp"]
  }

  subject {
    type     = "country"
    required = ["US"]
  }

  sans {
    type     = "dns_name"
    allowed  = ["*.example.com", "*.internal.example.com"]
    required = ["*.example.com"]
  }

  sans {
    type    = "email"
    allowed = ["*@example.com"]
  }

  key_usages {
    allowed = ["digital_signature", "key_encipherment"]
  }

  extended_key_usages {
    allowed = ["server_auth", "client_auth"]
  }

  validity {
    max = "90d"
  }

  algorithms {
    signature     = ["SHA256-RSA", "SHA256-ECDSA", "SHA384-ECDSA"]
    key_algorithm = ["RSA-2048", "RSA-3072", "ECDSA-P256", "ECDSA-P384"]
  }
}

# API enrollment profile for web server certificates
resource "infisical_cert_manager_certificate_profile" "web_server_api" {
  project_slug            = infisical_project.pki.slug
  ca_id                   = infisical_cert_manager_internal_ca_intermediate.issuing.id
  certificate_template_id = infisical_cert_manager_certificate_template.web_server.id

  slug            = "web-server-api"
  description     = "API enrollment for web server certificates"
  enrollment_type = "api"
  issuer_type     = "ca"

  api_config {
    auto_renew        = true
    renew_before_days = 7
  }
}

# EST enrollment profile for web server certificates
resource "infisical_cert_manager_certificate_profile" "web_server_est" {
  project_slug            = infisical_project.pki.slug
  ca_id                   = infisical_cert_manager_internal_ca_intermediate.issuing.id
  certificate_template_id = infisical_cert_manager_certificate_template.web_server.id

  slug            = "web-server-est"
  description     = "EST enrollment for web server certificates"
  enrollment_type = "est"
  issuer_type     = "ca"

  est_config {
    passphrase                      = var.est_passphrase
    disable_bootstrap_ca_validation = false
    ca_chain                        = var.encrypted_ca_chain
  }
}

# Self-signed profile for development
resource "infisical_cert_manager_certificate_profile" "self_signed_dev" {
  project_slug            = infisical_project.pki.slug
  certificate_template_id = infisical_cert_manager_certificate_template.web_server.id

  slug            = "self-signed-dev"
  description     = "Self-signed certificates for development"
  enrollment_type = "api"
  issuer_type     = "self-signed"

  api_config {
    auto_renew        = false
    renew_before_days = 7
  }
}

# ACME profile
resource "infisical_cert_manager_certificate_profile" "acme_profile" {
  project_slug            = infisical_project.pki.slug
  ca_id                   = var.acme_ca_id # Reference to existing ACME CA
  certificate_template_id = infisical_cert_manager_certificate_template.web_server.id

  slug            = "acme-letsencrypt"
  description     = "Let's Encrypt ACME certificates"
  enrollment_type = "acme"
  issuer_type     = "ca"
}

# ADCS profile (requires external ADCS CA to be configured)
resource "infisical_cert_manager_certificate_profile" "adcs_profile" {
  project_slug            = infisical_project.pki.slug
  ca_id                   = var.adcs_ca_id # Reference to existing ADCS CA
  certificate_template_id = infisical_cert_manager_certificate_template.web_server.id

  slug            = "adcs-corporate"
  description     = "Corporate ADCS certificates"
  enrollment_type = "api"
  issuer_type     = "ca"

  api_config {
    auto_renew        = true
    renew_before_days = 14
  }

  external_configs {
    template = "WebServerTemplate" # ADCS template name
  }
}

# Variables for sensitive values
variable "est_passphrase" {
  description = "Passphrase for EST enrollment"
  type        = string
  sensitive   = true
}

variable "encrypted_ca_chain" {
  description = "Encrypted CA certificate chain for EST enrollment"
  type        = string
  sensitive   = true
  default     = null
}

variable "acme_ca_id" {
  description = "ID of the ACME CA resource"
  type        = string
  default     = null
}

variable "adcs_ca_id" {
  description = "ID of the ADCS CA resource"
  type        = string
  default     = null
}