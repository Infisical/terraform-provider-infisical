terraform {
  required_providers {
    infisical = {
      # version = <latest version>
      source = "infisical/infisical"
    }
  }
}

provider "infisical" {
  host = "https://app.infisical.com" # Only required if using self hosted instance of Infisical, default is https://app.infisical.com
  auth = {
    universal = {
      client_id     = "<machine-identity-client-id>"
      client_secret = "<machine-identity-client-secret>"
    }
  }
}

resource "infisical_dynamic_secret_openrouter" "openrouter" {
  name             = "openrouter-dynamic-secret-example"
  project_slug     = "your-project-slug"
  environment_slug = "dev"
  path             = "/"
  default_ttl      = "1h"
  max_ttl          = "24h"

  configuration = {
    management_key        = "<your-openrouter-management-key>"
    workspace_id          = "<your-openrouter-workspace-id>" # Optional, defaults to the management key's default workspace
    limit                 = "2.5"                            # Optional, spending limit in USD per created API key
    limit_reset           = "monthly"                        # Optional, one of daily, weekly or monthly
    include_byok_in_limit = false
  }

  username_template = "{{randomUsername}}"
}
