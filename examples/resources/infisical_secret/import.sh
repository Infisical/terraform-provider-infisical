# Import by Infisical secret UUID. The secret value is written to state under `value`.
terraform import infisical_secret.example <secret_id>

# Import by workspace, environment, folder path, and secret name. The secret value is written to state under `value`.
terraform import infisical_secret.example '<workspace_id>:<env_slug>:<folder_path>:<secret_name>'

# Import as write-only by prefixing the ID with `write-only:`. The secret value is NOT written to state; `value_wo_version` is initialized to 1 so a config with `value_wo_version = 1` will not trigger a spurious update on the first plan.
terraform import infisical_secret.example 'write-only:<secret_id>'

# The write-only prefix also works with the composite format.
terraform import infisical_secret.example 'write-only:<workspace_id>:<env_slug>:<folder_path>:<secret_name>'
