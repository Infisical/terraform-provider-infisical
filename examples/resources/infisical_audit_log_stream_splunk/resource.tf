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

# Stream every audit log to a Splunk Cloud HTTP Event Collector. Splunk Cloud serves the
# collector on port 443 from a dedicated hostname, which is not the one you use for the web
# interface: http-inputs-<stack>.splunkcloud.com on AWS, http-inputs.<stack>.splunkcloud.com
# on GCP and Azure.
resource "infisical_audit_log_stream_splunk" "siem" {
  credentials = {
    hostname = "http-inputs-acme.splunkcloud.com"
    port     = 443
    token    = "<your-hec-token>"
  }
}

# Splunk Enterprise defaults to port 8088. This stream is scoped, so only the listed products
# reach it; omit the filters block to stream everything.
resource "infisical_audit_log_stream_splunk" "secrets_only" {
  credentials = {
    hostname = "splunk.internal.example.com"
    token    = "<your-hec-token>"
  }

  filters = {
    products = ["secret-manager", "organization"]
  }
}
