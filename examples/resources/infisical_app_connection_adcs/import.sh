# Import a Microsoft ADCS app connection by its ID.
# The API never returns the password, so the first apply after an import sends the configured credentials again.
terraform import infisical_app_connection_adcs.adcs <app-connection-id>
