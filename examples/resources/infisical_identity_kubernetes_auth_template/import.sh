terraform import infisical_identity_kubernetes_auth_template.example <template_id>

# token_reviewer_jwt is write-only, so an imported template has it empty. has_token_reviewer_jwt
# still reports whether Infisical holds one.
