# The get-by-id endpoint does not return the folder scope, so import takes the scope plus the name
terraform import infisical_proxied_service.example <project_id>,<environment_slug>,<secret_path>,<name>

# For example
terraform import infisical_proxied_service.github 92ac1b1a-1b1a-4b1a-9b1a-1b1a4b1a9b1a,dev,/coding-agent,github
