terraform import infisical_gateway.example <gateway_id>

# token_reviewer_jwt is write-only, so an imported Kubernetes gateway has it empty.
# Gateways bound to a machine identity predate auth methods and cannot be imported.
