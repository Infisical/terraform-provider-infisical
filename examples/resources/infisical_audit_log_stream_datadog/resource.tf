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

# Use the log intake URL for your Datadog site, e.g. https://http-intake.logs.datadoghq.eu
# for EU1.
resource "infisical_audit_log_stream_datadog" "observability" {
  credentials = {
    url   = "https://http-intake.logs.datadoghq.com/api/v2/logs"
    token = "<your-datadog-api-key>"
  }
}
