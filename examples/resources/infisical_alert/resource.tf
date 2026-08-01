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

# Email the security team 30 days before an organization level machine identity's
# authentication credentials expire
resource "infisical_alert" "ci_identity" {
  resource_type = "identity.authentication"
  resource_id   = "<your-machine-identity-id>" # A machine identity, for this resource type
  name          = "CI identity credentials expiring"
  description   = "Rotate the CI machine identity client secret before it expires"

  # The condition block is what the alert fires on, and every alert carries exactly one.
  authentication_expiry = {
    alert_before_days = 30
  }

  # Each channel is keyed by its name, and the block it carries is what gives it its type.
  # Renaming a channel deletes it and creates a new one, so the new channel notifies about
  # everything that is still expiring, even if the old one already did.
  channels = {
    "Security team" = {
      email = {
        recipients = [
          {
            type = "user"
            id   = "<your-user-id>"
          },
          {
            type = "group"
            id   = "<your-group-id>"
          },
        ]
      }
    }
  }
}

# Remind a project level machine identity's owners every day for the last week, over
# Slack, a signed webhook and PagerDuty
resource "infisical_alert" "deploy_identity" {
  resource_type = "identity.authentication"
  resource_id   = "<your-machine-identity-id>"
  project_id    = "<your-project-id>" # Required for resources that belong to a project
  name          = "Deploy identity credentials expiring"

  authentication_expiry = {
    alert_before_days = 7
    daily_reminder    = true
  }

  channels = {
    "Platform Slack" = {
      slack = {
        webhook_url = "https://hooks.slack.com/services/<your-slack-webhook-path>"
      }
    }

    "Internal automation" = {
      webhook = {
        url            = "https://example.com/infisical-alerts"
        signing_secret = "<signing-secret>" # Optional: used to sign the payload so the receiver can verify it
      }
    }

    "On-call" = {
      enabled = false # Configured up front, but not paging anyone yet
      pagerduty = {
        integration_key = "<your-pagerduty-events-api-v2-integration-key>"
      }
    }
  }
}
