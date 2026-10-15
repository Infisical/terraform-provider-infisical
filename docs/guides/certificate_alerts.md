---
page_title: "Certificate alerts"
subcategory: "Certificate Management"
description: |-
  Use infisical_alert to get notified about certificates in Certificate Manager, either in one application or across Certificate Manager.
---

# Certificate alerts

Certificate Manager alerts are managed with the [`infisical_alert`](../resources/alert) resource. Like in Infisical, an alert either watches one application or all of Certificate Manager, and fires on one event, which you pick with exactly one condition block:

- `expiry` fires before a certificate expires. Set `alert_before_days` between 1 and 365, and `daily_reminder = true` to keep notifying every day until it expires.
- `issuance`, `renewal` and `revocation` fire when a certificate is issued, renewed or revoked. They take no settings, so write them as `issuance = {}`.

## Alerts on an application

Set `resource_type = "cert-manager.application"` and `resource_id` to the application's ID. The alert covers every certificate in the application, and an application can have one alert per event.

| Event | Condition block |
|---|---|
| Certificate Expiration | `expiry` |
| Certificate Issuance | `issuance` |
| Certificate Renewal | `renewal` |
| Certificate Revocation | `revocation` |

```terraform
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
```

## Alerts across Certificate Manager

These alerts watch every certificate in Certificate Manager, and you can create as many as you need for the same event, for example to send expiring certificates from different profiles to different teams. Only Certificate Manager admins can create them.

| Event | `resource_type` | Condition block |
|---|---|---|
| Certificate Expiration | `cert-manager` | `expiry` |
| Certificate Issuance | `cert-manager` | `issuance` |
| Certificate Renewal | `cert-manager` | `renewal` |
| Certificate Revocation | `cert-manager` | `revocation` |
| Signer Certificate Expiration (Code Signing) | `cert-manager.signer` | `expiry` |

### Certificate events

Optionally narrow a `cert-manager` alert with `filters`. A certificate has to match every list that is set, and each list left out covers everything:

- `application_ids`: only certificates in these applications.
- `profile_ids`: only certificates issued from these certificate profiles.
- `sources`: only certificates with these sources (`issued`, `discovered` or `imported`).

```terraform
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
```

### Signer certificate expiration

Fires before the certificate any code signing signer signs with expires.

```terraform
# Warn before the certificate any code signing signer signs with expires
resource "infisical_alert" "signers_expiring" {
  resource_type = "cert-manager.signer"
  name          = "Signer certificates expiring"

  expiry = {
    alert_before_days = 90
  }

  channels = {
    release_team = {
      name = "Release team"
      email = {
        recipients = [{ type = "group", id = "<your-group-id>" }]
      }
    }
  }
}
```

## Channels and recipients

Each alert has up to 10 channels, keyed by a name of your choosing: `email`, `slack`, `webhook` or `pagerduty`. Email channels notify `user` and `group` recipients, or an email address directly with `type = "email"`, as long as it is on one of your organization's verified email domains. Slack, webhook and PagerDuty channels need a plan that includes them.

## Changes made in Infisical

Terraform picks up changes made to these alerts in Infisical on the next plan and puts the configuration back on apply. A duration saved in Infisical as weeks, months or years, such as `2w` or `1m`, reads back as the same number of days (`14`, `30`), so it shows no change. An alert deleted in Infisical, or removed along with its application, is created again on the next apply.

## Import

```shell
terraform import infisical_alert.example <alert_id>
```

An imported alert keys its channels by their names in Infisical, so use those names as the channel keys in your configuration.
