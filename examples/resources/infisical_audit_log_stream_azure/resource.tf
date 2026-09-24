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

# Streams into an Azure Monitor custom log table, which is how audit logs reach Microsoft
# Sentinel. The app registration needs the Monitoring Metrics Publisher role on the Data
# Collection Rule.
resource "infisical_audit_log_stream_azure" "sentinel" {
  credentials = {
    tenant_id     = "<your-entra-tenant-id>"
    client_id     = "<your-app-registration-client-id>"
    client_secret = "<your-app-registration-client-secret>"
    dce_url       = "https://<your-dce>.<region>.ingest.monitor.azure.com"
    dcr_id        = "dcr-<32-hex-characters>"
    clt_name      = "InfisicalAuditLogs_CL"
  }
}
