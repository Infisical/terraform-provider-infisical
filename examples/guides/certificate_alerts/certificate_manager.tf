# Alert on certificates issued anywhere in Certificate Manager, narrowed with filters.
# A certificate has to match every list that is set.
resource "infisical_alert" "public_tls_issued" {
  resource_type = "cert-manager"
  name          = "Public TLS certificate issued"

  issuance = {}

  filters = {
    application_ids = ["<your-application-id>"]
    profile_ids     = ["<your-certificate-profile-id>"]
    sources         = ["issued"] # Options: issued, discovered, imported
  }

  channels = {
    security_slack = {
      name = "Security Slack"
      slack = {
        webhook_url = "https://hooks.slack.com/services/<your-slack-webhook-path>"
      }
    }
  }
}

# Without filters, the alert covers every certificate in Certificate Manager
resource "infisical_alert" "anything_expiring" {
  resource_type = "cert-manager"
  name          = "Any certificate expiring"

  expiry = {
    alert_before_days = 30
  }

  channels = {
    platform = {
      name = "Platform"
      webhook = {
        url            = "https://example.com/infisical-alerts"
        signing_secret = "<signing-secret>" # Optional: lets the receiver verify the payload
      }
    }
  }
}
