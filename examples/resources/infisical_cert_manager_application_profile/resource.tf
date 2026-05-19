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
  country       = "US"
  key_algorithm = "RSA_2048"
}

resource "infisical_cert_manager_certificate_policy" "web_server" {
  name = "web-server-policy"

  subject {
    type     = "common_name"
    allowed  = ["*.example.com"]
    required = ["*.example.com"]
  }

  key_usages {
    allowed = ["digital_signature", "key_encipherment"]
  }

  extended_key_usages {
    allowed = ["server_auth"]
  }

  validity {
    max = "90d"
  }

  algorithms {
    signature     = ["SHA256-RSA"]
    key_algorithm = ["RSA-2048"]
  }
}

resource "infisical_cert_manager_certificate_profile" "web_server" {
  ca_id                 = infisical_cert_manager_internal_ca.issuing.id
  certificate_policy_id = infisical_cert_manager_certificate_policy.web_server.id

  name            = "web-server"
  issuer_type     = "ca"

  defaults {
    common_name         = "service.example.com"
    ttl_days            = 90
    key_algorithm       = "RSA_2048"
    signature_algorithm = "RSA-SHA256"
    key_usages          = ["digital_signature"]
    extended_key_usages = ["server_auth"]
  }
}

resource "infisical_cert_manager_application" "platform" {
  name = "platform"
}

# Attach the profile to the application and configure its enrollment methods. Each
# enrollment block (api_config, est_config, acme_config, scep_config) is optional —
# add, remove, or edit a block to enable, disable, or change the corresponding enrollment.
# Only one resource per (application_id, profile_id) pair.
resource "infisical_cert_manager_application_profile" "platform_web_server" {
  application_id = infisical_cert_manager_application.platform.id
  profile_id     = infisical_cert_manager_certificate_profile.web_server.id

  api_config = {
    auto_renew        = true
    renew_before_days = 7
  }
}
