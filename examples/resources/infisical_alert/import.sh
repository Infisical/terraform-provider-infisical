terraform import infisical_alert.example <alert_id>

# An imported alert keys each of its channels by the name it has in Infisical, so write the
# channels in your configuration under those same keys, or the import deletes them and creates
# new ones that notify about everything still expiring, even though the imported channels
# already did. The keys are yours to choose from then on, since only the name attribute reaches
# Infisical.
#
# Write-only channel secrets (a Slack webhook URL, a PagerDuty integration key, a webhook
# signing secret) are never returned by the API, so the first plan after an import shows them
# being set. A signing secret left out of the configuration keeps whatever is stored.
