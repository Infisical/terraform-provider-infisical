# Warn before the certificate of a chosen code signing signer expires.
# Leave signer_ids out to cover every signer.
resource "infisical_alert" "signers_expiring" {
  resource_type = "cert-manager.signer"
  name          = "Signer certificates expiring"
  event_type    = "certificate.expiry"

  condition = {
    alert_before_days = 90
    signer_ids        = ["<your-signer-id>"]
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
