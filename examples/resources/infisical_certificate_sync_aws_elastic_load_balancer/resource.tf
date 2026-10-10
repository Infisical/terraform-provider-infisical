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

resource "infisical_certificate_sync_aws_elastic_load_balancer" "example" {
  name           = "aws-elb-certificate-sync-demo"
  description    = "Demo of AWS Elastic Load Balancer certificate sync"
  application_id = "<cert-manager-application-id>"
  connection_id  = "<app-connection-id>"

  certificate_filters = {
    certificate_ids = [infisical_cert_manager_certificate.example.id]
  }

  default_certificate_id = infisical_cert_manager_certificate.example.id

  sync_options = {
    certificate_name_schema = "Infisical-{{certificateId}}"
    can_remove_certificates = false
    include_root_ca         = false
    preserve_arn            = true
  }

  destination_config = {
    aws_region        = "us-east-1"
    load_balancer_arn = "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-load-balancer/50dc6c495c0c9188"
    listeners = [
      {
        listener_arn = "arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/my-load-balancer/50dc6c495c0c9188/f2f7dc8efc522ab2"
        port         = 443
        protocol     = "HTTPS"
      }
    ]
  }
}
