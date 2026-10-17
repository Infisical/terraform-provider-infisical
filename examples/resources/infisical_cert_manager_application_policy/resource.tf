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

resource "infisical_cert_manager_application" "payments" {
  name = "payments"
}

# Approvers of an application policy, users and groups alike, have to be members of the application
resource "infisical_cert_manager_application_user" "security_lead" {
  application_id = infisical_cert_manager_application.payments.id
  email          = "security-lead@example.com"
  role           = "admin"
}

resource "infisical_cert_manager_application_group" "platform" {
  application_id = infisical_cert_manager_application.payments.id
  group_id       = "<platform-group-id>"
  role           = "operator"
}

# Certificates requested through the payments application from the public TLS profile need the
# security lead's approval, then one approval from the platform group
resource "infisical_cert_manager_application_policy" "payments_public_tls" {
  name           = "Payments public TLS"
  application_id = infisical_cert_manager_application.payments.id
  profile_names  = ["public-tls"] # The name of an infisical_cert_manager_certificate_profile

  bypass_for_machine_identities = true # Automation requesting these certificates skips the approval
  max_request_ttl               = "7d" # A request nobody approves within a week expires

  steps = [
    {
      name               = "Security review"
      required_approvals = 1
      approvers = [
        { type = "user", id = "<security-lead-user-id>" },
      ]
    },
    {
      name               = "Platform sign-off"
      required_approvals = 1
      notify_approvers   = false # Approvers see the request in Infisical, but are not emailed
      approvers = [
        { type = "group", id = infisical_cert_manager_application_group.platform.group_id },
      ]
    },
  ]

  depends_on = [
    infisical_cert_manager_application_user.security_lead,
    infisical_cert_manager_application_group.platform,
  ]
}
