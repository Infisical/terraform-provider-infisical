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

resource "infisical_certificate_sync_aws_secrets_manager" "example" {
  name           = "aws-secrets-manager-certificate-sync-demo"
  description    = "Demo of AWS Secrets Manager certificate sync"
  application_id = "<cert-manager-application-id>"
  connection_id  = "<app-connection-id>"

  certificate_filters = {
    certificate_ids = [infisical_cert_manager_certificate.example.id]
  }

  sync_options = {
    certificate_name_schema      = "infisical-{{certificateId}}" # Must include {{certificateId}} or {{shortCertificateId}}
    can_remove_certificates      = true
    include_root_ca              = false
    preserve_secret_on_renewal   = true
    update_existing_certificates = true
    field_mappings = {
      certificate       = "certificate"
      private_key       = "private_key"
      certificate_chain = "certificate_chain"
      ca_certificate    = "ca_certificate"
    }
  }

  destination_config = {
    aws_region = "us-east-1"
    kms_key_id = "alias/my-secrets-key" # Optional, defaults to the AWS managed key
  }
}
