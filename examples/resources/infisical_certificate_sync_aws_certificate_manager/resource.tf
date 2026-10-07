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

resource "infisical_certificate_sync_aws_certificate_manager" "example" {
  name           = "aws-certificate-manager-certificate-sync-demo"
  description    = "Demo of AWS Certificate Manager certificate sync"
  application_id = "<cert-manager-application-id>"
  connection_id  = "<app-connection-id>"

  # Omit certificate_filters to choose the synced certificates outside Terraform.
  # Fields are combined: a certificate must match every field that is set.
  certificate_filters = {
    certificate_ids = [
      infisical_cert_manager_certificate.api.id,
      infisical_cert_manager_certificate.www.id,
    ]
    # profile_ids = ["<certificate-profile-id>"]
    # metadata    = [{ key = "env", value = "prod" }]
  }

  sync_options = {
    certificate_name_schema = "Infisical-{{certificateId}}" # Must include {{certificateId}} or {{shortCertificateId}}
    can_remove_certificates = true
    include_root_ca         = false
    preserve_arn            = true
  }

  destination_config = {
    aws_region = "<aws-region>" # e.g. us-east-1
  }
}
