terraform import infisical_audit_log_stream_azure.example <audit_log_stream_id>

# The API never returns the client secret, so the first plan after an import shows it being set.
# Everything else (tenant, client, DCE, DCR, table name) is imported as configured in Infisical.
