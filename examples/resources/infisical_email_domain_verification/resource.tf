terraform {
  required_providers {
    infisical = {
      # version = <latest version>
      source = "infisical/infisical"
    }
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 5.0"
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

# 1. Claim the email domain.
resource "infisical_email_domain" "example" {
  domain = "example.com"
}

# 2. Publish the DNS record. Any DNS provider works; this example uses Cloudflare.
resource "cloudflare_dns_record" "infisical_verification" {
  zone_id = "<your-cloudflare-zone-id>"
  type    = infisical_email_domain.example.verification_record.type
  name    = infisical_email_domain.example.verification_record.name
  content = infisical_email_domain.example.verification_record.value
  ttl     = 300
}

# 3. Verify the domain. Create retries while the DNS record propagates.
resource "infisical_email_domain_verification" "example" {
  email_domain_id = infisical_email_domain.example.id

  timeouts = {
    create = "15m"
  }

  depends_on = [cloudflare_dns_record.infisical_verification]
}
