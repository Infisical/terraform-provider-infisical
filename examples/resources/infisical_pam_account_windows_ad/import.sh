# This will import the Windows Active Directory PAM account by its ID.
# Secrets are never read back, so set the credentials in the configuration after importing.
terraform import infisical_pam_account_windows_ad.example <account_id>
