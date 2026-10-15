resource "infisical_cert_manager_application" "payments" {
  name = "payments"
}

# Email the payments team two months before any certificate in the application expires
resource "infisical_alert" "payments_expiring" {
  resource_type = "cert-manager.application"
  resource_id   = infisical_cert_manager_application.payments.id
  name          = "Payments certificates expiring"

  expiry = {
    alert_before_days = 60
    daily_reminder    = true # Keep reminding every day until the certificate expires
  }

  channels = {
    payments_team = {
      name = "Payments team"
      email = {
        recipients = [
          { type = "user", id = "<your-user-id>" },
          { type = "email", id = "payments-oncall@example.com" }, # Must be on a verified domain
        ]
      }
    }
  }
}

# Page on-call whenever a certificate in the application is revoked
resource "infisical_alert" "payments_revoked" {
  resource_type = "cert-manager.application"
  resource_id   = infisical_cert_manager_application.payments.id
  name          = "Payments certificate revoked"

  revocation = {}

  channels = {
    on_call = {
      name = "On-call"
      pagerduty = {
        integration_key = "<your-pagerduty-events-api-v2-integration-key>"
      }
    }
  }
}
