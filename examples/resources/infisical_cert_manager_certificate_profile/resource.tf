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

resource "infisical_cert_manager_certificate_policy" "web_server" {
  name        = "web-server-policy"
  description = "Policy for web server certificates"

  subject {
    type     = "common_name"
    allowed  = ["*.example.com"]
    required = ["*.example.com"]
  }

  subject {
    type     = "organization"
    required = ["Example Corp"]
  }

  sans {
    type     = "dns_name"
    allowed  = ["*.example.com"]
    required = ["*.example.com"]
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
    signature     = ["SHA256-RSA"]
    key_algorithm = ["RSA-2048", "RSA-3072"]
  }
}

resource "infisical_cert_manager_certificate_profile" "web_server" {
  ca_id                 = infisical_cert_manager_internal_ca.issuing.id
  certificate_policy_id = infisical_cert_manager_certificate_policy.web_server.id

  name        = "web-server"
  description = "Profile for issuing web server certificates"
  issuer_type = "ca"

  defaults {
    common_name         = "service.example.com"
    ttl_days            = 90
    key_algorithm       = "RSA_2048"
    signature_algorithm = "RSA-SHA256"
    key_usages          = ["digital_signature"]
    extended_key_usages = ["server_auth"]
  }
}

resource "infisical_cert_manager_certificate_profile" "self_signed_dev" {
  certificate_policy_id = infisical_cert_manager_certificate_policy.web_server.id

  name        = "self-signed-dev"
  description = "Self-signed certificates for development"
  issuer_type = "self-signed"

  defaults {
    common_name         = "dev.example.com"
    ttl_days            = 90
    key_algorithm       = "RSA_2048"
    signature_algorithm = "RSA-SHA256"
    key_usages          = ["digital_signature"]
    extended_key_usages = ["server_auth"]
  }
}
