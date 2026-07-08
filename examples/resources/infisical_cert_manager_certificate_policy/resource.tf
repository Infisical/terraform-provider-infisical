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

resource "infisical_cert_manager_certificate_policy" "web_server" {
  name        = "web-server-policy"
  description = "Policy for web server certificates"

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

  sans {
    type    = "ip_address"
    allowed = ["10.0.0.0/8", "192.168.0.0/16"]
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

  # Leaf certificates must not be CA certificates
  basic_constraints {
    is_ca = "denied"
  }
}

resource "infisical_cert_manager_certificate_policy" "intermediate_ca" {
  name        = "intermediate-ca-policy"
  description = "Policy for intermediate CA certificates"

  subject {
    type     = "common_name"
    required = ["Example Corp Intermediate CA"]
  }

  subject {
    type     = "organization"
    required = ["Example Corp"]
  }

  subject {
    type    = "organizational_unit"
    allowed = ["PKI", "Security"]
  }

  subject {
    type     = "country"
    required = ["US"]
  }

  subject {
    type    = "state"
    allowed = ["California"]
  }

  subject {
    type    = "locality"
    allowed = ["San Francisco"]
  }

  key_usages {
    required = ["key_cert_sign", "crl_sign"]
  }

  validity {
    max = "5y"
  }

  algorithms {
    signature     = ["SHA256-RSA", "SHA384-ECDSA"]
    key_algorithm = ["RSA-4096", "ECDSA-P384"]
  }

  # Issued certificates must be CA certificates, limited to one further level of subordinate CAs
  basic_constraints {
    is_ca           = "required"
    max_path_length = 1
  }
}

resource "infisical_cert_manager_certificate_policy" "code_signing" {
  name        = "code-signing-policy"
  description = "Policy for code signing certificates"

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

  key_usages {
    allowed = ["digital_signature"]
  }

  extended_key_usages {
    allowed = ["code_signing"]
  }

  validity {
    max = "1y"
  }

  algorithms {
    signature     = ["SHA256-RSA", "SHA256-ECDSA"]
    key_algorithm = ["RSA-2048", "RSA-3072", "ECDSA-P256"]
  }
}
