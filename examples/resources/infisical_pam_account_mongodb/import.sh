# This will import the MongoDB PAM account by its ID.
# Secrets are never read back, so set the password in the configuration after importing.
terraform import infisical_pam_account_mongodb.example <account_id>
