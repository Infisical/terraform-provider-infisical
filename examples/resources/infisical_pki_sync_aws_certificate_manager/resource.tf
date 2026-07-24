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

resource "infisical_pki_sync_aws_certificate_manager" "example" {
  name           = "aws-certificate-manager-pki-sync-demo"
  description    = "Demo of AWS Certificate Manager PKI sync"
  application_id = "<cert-manager-application-id>"
  connection_id  = "<app-connection-id>"

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
