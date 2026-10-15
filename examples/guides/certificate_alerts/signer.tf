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
