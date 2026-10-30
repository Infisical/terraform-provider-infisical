# This will import the Kubernetes PAM account by its ID.
# Secrets are never read back, so set the credentials in the configuration after importing.
terraform import infisical_pam_account_kubernetes.example <account_id>
