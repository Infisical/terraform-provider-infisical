# This will import the PAM folder approval policy by its ID.
# GET /api/v1/approval-policies/pam-access?scope=pam-folder&id=<folder_id> returns the ID of a folder's policy.
terraform import infisical_pam_folder_approval_policy.example <policy_id>
