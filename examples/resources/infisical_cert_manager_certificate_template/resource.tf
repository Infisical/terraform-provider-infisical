resource "infisical_project" "pki" {
  name        = "PKI Project"
  slug        = "pki-project"
  description = "Project for PKI certificate management"
  type        = "cert-manager"
}

resource "infisical_cert_manager_certificate_template" "web_server" {
  project_slug = infisical_project.pki.slug

  name        = "web-server-template"
  description = "Template for web server certificates"

  # Subject Attribute Policies
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

  # Subject Alternative Name Policies
  sans {
    type     = "dns_name"
    allowed  = ["*.example.com", "*.internal.example.com"]
    required = ["*.example.com"]
  }

  sans {
    type    = "email"
    allowed = ["*@example.com"]
  }

  sans {
    type    = "ip_address"
    allowed = ["10.0.0.0/8", "192.168.0.0/16"]
  }

  # Certificate Properties
  key_usages {
    allowed = ["digital_signature", "key_encipherment"]
  }

  extended_key_usages {
    allowed = ["server_auth", "client_auth"]
  }

  # Validity Constraints
  validity {
    max = "90d" # Uses simplified format: 90d, 12m, 1y, 24h
  }

  # Algorithm Constraints
  algorithms {
    signature     = ["SHA256-RSA", "SHA256-ECDSA", "SHA384-ECDSA"]
    key_algorithm = ["RSA-2048", "RSA-3072", "ECDSA-P256", "ECDSA-P384"]
  }
}

resource "infisical_cert_manager_certificate_template" "code_signing" {
  project_slug = infisical_project.pki.slug

  name        = "code-signing-template"
  description = "Template for code signing certificates"

  # Subject Attribute Policies
  subject {
    type     = "common_name"
    allowed  = ["Example Corp Code Signing"]
    required = ["Example Corp Code Signing"]
  }

  subject {
    type     = "organization"
    required = ["Example Corp"]
  }

  subject {
    type     = "country"
    required = ["US"]
  }

  # Certificate Properties
  key_usages {
    allowed = ["digital_signature"]
  }

  extended_key_usages {
    allowed = ["code_signing"]
  }

  # Validity Constraints
  validity {
    max = "1y"
  }

  # Algorithm Constraints
  algorithms {
    signature     = ["SHA256-RSA", "SHA256-ECDSA"]
    key_algorithm = ["RSA-2048", "RSA-3072", "ECDSA-P256"]
  }
}