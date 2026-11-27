# This will import the Microsoft ADCS app connection by its ID.
# Secrets are never read back, so the first apply after importing sends the credentials in the configuration.
terraform import infisical_app_connection_adcs.example <app_connection_id>
