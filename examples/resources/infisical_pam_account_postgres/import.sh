# This will import the PostgreSQL PAM account by its ID.
# Secrets are never read back, so set the credentials in the configuration after importing.
terraform import infisical_pam_account_postgres.example <account_id>
