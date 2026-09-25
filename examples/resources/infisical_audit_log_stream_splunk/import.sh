terraform import infisical_audit_log_stream_splunk.example <audit_log_stream_id>

# The API never returns credential secrets, so the first plan after an import shows the HEC
# token being set. Readable fields (hostname, port) are imported as they are configured in
# Infisical.
