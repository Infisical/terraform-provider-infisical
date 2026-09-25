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

# Create an HTTP Logs and Metrics Source in Sumo Logic and choose the "Auth Header" option when
# generating its URL. Sumo Logic then shows a base URL and an x-sumo-token header: the token
# from that header is what goes below.
resource "infisical_audit_log_stream_sumo_logic" "siem" {
  credentials = {
    url   = "https://endpoint4.collection.sumologic.com/receiver/v1/http"
    token = "<your-sumo-logic-source-token>"
  }
}
