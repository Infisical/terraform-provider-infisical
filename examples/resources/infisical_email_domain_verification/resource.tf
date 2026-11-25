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

resource "infisical_email_domain" "example" {
  domain = "example.com"
}

# Publish a DNS TXT record before the verification runs, with your DNS provider's resource:
#   name:  infisical_email_domain.example.verification_record_name
#   value: infisical_email_domain.example.verification_code
# Add that resource to depends_on below, so Terraform creates the record first.

resource "infisical_email_domain_verification" "example" {
  email_domain_id = infisical_email_domain.example.id

  # Verification retries while the DNS record propagates.
  timeouts = {
    create = "15m"
  }
}
