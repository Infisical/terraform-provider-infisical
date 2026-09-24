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

# Any endpoint that accepts a JSON array of audit log events. Batches are bounded at 500 events
# or 1 MB per request, so the receiver has to handle an array rather than a single object.
resource "infisical_audit_log_stream_custom" "better_stack" {
  credentials = {
    url = "https://in.logs.betterstack.com"
    headers = {
      Authorization = "Bearer <your-source-token>"
    }
  }

  filters = {
    products = ["organization"]
  }
}
