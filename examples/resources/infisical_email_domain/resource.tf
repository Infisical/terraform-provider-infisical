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

# Publish these values as a DNS TXT record, then verify the domain with infisical_email_domain_verification.
output "infisical_txt_record_name" {
  value = infisical_email_domain.example.verification_record_name
}

output "infisical_txt_record_value" {
  value = infisical_email_domain.example.verification_code
}
