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

resource "infisical_audit_log_stream_cribl" "pipeline" {
  credentials = {
    url   = "https://<your-cribl-worker-host>:10080/cribl/_bulk"
    token = "<your-cribl-http-source-token>"
  }
}
