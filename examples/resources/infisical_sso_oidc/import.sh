# This will import the OIDC configuration by the ID of the organization. An organization has at most one OIDC configuration.
# The client secret is written to state under `client_secret`.
terraform import infisical_sso_oidc.example <organization_id>

# For a configuration that uses `client_secret_wo`, prefix the ID with `write-only:`. The client secret is NOT written to state;
terraform import infisical_sso_oidc.example 'write-only:<organization_id>'
