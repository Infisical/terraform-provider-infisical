terraform {
  required_providers {
    infisical = {
      source = "infisical/infisical"
    }
  }
}

provider "infisical" {
  host          = "https://app.infisical.com"
  client_id     = var.client_id
  client_secret = var.client_secret
}

resource "infisical_cert_manager_internal_ca" "root" {
  type          = "root"
  name          = "enterprise-root-ca"
  common_name   = "Enterprise Root Certificate Authority"
  organization  = "Acme Corp"
  ou            = "IT Security"
  country       = "US"
  province      = "California"
  locality      = "San Francisco"
  key_algorithm = "RSA_2048"
}

resource "infisical_cert_manager_internal_ca" "issuing" {
  type          = "intermediate"
  name          = "enterprise-issuing-ca"
  common_name   = "Enterprise Issuing Certificate Authority"
  organization  = "Acme Corp"
  ou            = "IT Security"
  country       = "US"
  province      = "California"
  locality      = "San Francisco"
  key_algorithm = "RSA_2048"
}

# The signing key is generated and kept in the HSM.
# Find the HSM Connector ID under Certificate Manager > Settings > HSM Connectors.
resource "infisical_cert_manager_internal_ca" "hsm_root" {
  type             = "root"
  name             = "hsm-root-ca"
  common_name      = "HSM Root Certificate Authority"
  organization     = "Acme Corp"
  key_algorithm    = "EC_secp384r1"
  key_source       = "hsm"
  hsm_connector_id = "<your-hsm-connector-id>"
}

# The signing key is generated in AWS KMS, and AWS KMS makes every signature.
resource "infisical_cert_manager_internal_ca" "aws_kms_root" {
  type              = "root"
  name              = "aws-kms-root-ca"
  common_name       = "AWS KMS Root Certificate Authority"
  organization      = "Acme Corp"
  key_algorithm     = "RSA_4096"
  key_source        = "aws-kms"
  app_connection_id = "<your-aws-app-connection-id>"
  region            = "us-east-1"
}
