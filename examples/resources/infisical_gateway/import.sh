terraform import infisical_gateway.example <gateway_id>

# token_reviewer_jwt is write-only: Infisical never returns it, so an imported Kubernetes gateway
# has it empty and the first plan afterwards shows it being set. Leaving it out of the
# configuration keeps whatever is stored.
#
# Gateways created before auth methods existed are bound to a machine identity instead. Those
# cannot be imported, because there is no way to manage or migrate their auth method. Create a new
# gateway with aws_auth, gcp_auth, kubernetes_auth or token_auth and repoint the deployment at it.
