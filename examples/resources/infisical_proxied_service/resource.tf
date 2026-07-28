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

# Secret substitution: the agent receives GITHUB_TOKEN set to a fake ghp_ value and uses it
# normally. The Agent Proxy swaps it for the real GITHUB_PAT secret on the way out.
resource "infisical_proxied_service" "github" {
  name         = "github"
  project_id   = "<project-id>"
  environment  = "<environment-slug>"
  secret_path  = "/coding-agent"
  host_pattern = "api.github.com"

  credentials = [
    {
      secret_key            = "GITHUB_PAT"
      role                  = "credential-substitution"
      placeholder_key       = "GITHUB_TOKEN"
      placeholder_value     = "ghp_0000000000000000000000000000000000"
      substitution_surfaces = ["header"]
    }
  ]
}

# Header rewriting: the agent sends no credential at all, and the Agent Proxy sets the
# Authorization header on matching requests.
resource "infisical_proxied_service" "internal_api" {
  name         = "internal-api"
  project_id   = "<project-id>"
  environment  = "<environment-slug>"
  secret_path  = "/coding-agent"
  host_pattern = "api.internal.example.com,*.staging.example.com:8443"

  credentials = [
    {
      secret_key    = "INTERNAL_API_KEY"
      role          = "header-rewrite"
      header_name   = "Authorization"
      header_prefix = "Bearer" # the proxy inserts the space before the value
    }
  ]
}

# The agent identity only needs Proxy on proxied services: it routes traffic and has
# credentials applied, without being able to read any secret value.
resource "infisical_project_identity_specific_privilege" "agent" {
  project_slug = "<project-slug>"
  identity_id  = "<agent-identity-id>"
  permissions_v2 = [
    {
      action  = ["proxy"]
      subject = "proxied-services"
    },
  ]
}

# The Agent Proxy identity reads the real values and reports usage back to Infisical.
resource "infisical_project_identity_specific_privilege" "agent_proxy" {
  project_slug = "<project-slug>"
  identity_id  = "<agent-proxy-identity-id>"
  permissions_v2 = [
    {
      action  = ["describeSecret", "readValue"]
      subject = "secrets"
    },
    {
      action  = ["report-usage"]
      subject = "proxied-services"
    },
  ]
}
