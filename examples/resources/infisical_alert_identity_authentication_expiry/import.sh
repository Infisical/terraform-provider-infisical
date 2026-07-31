terraform import infisical_alert_identity_authentication_expiry.example <alert_id>

# An imported alert keys each of its channels by the name it has in Infisical, so write the
# channels in your configuration under those same names. Keying them differently renames the
# channels, which deletes them and creates new ones that notify about everything still
# expiring, even though the imported channels already did.
#
# Write-only channel secrets (a Slack webhook URL, a PagerDuty integration key, a webhook
# signing secret) are never returned by the API, so the first plan after an import shows them
# being set. A signing secret left out of the configuration keeps whatever is stored.
