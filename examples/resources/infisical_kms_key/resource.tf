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

# Create an encryption KMS key
resource "infisical_kms_key" "encryption_key" {
  project_id           = "<your-project-id>"
  name                 = "my-encryption-key"
  description          = "KMS key for encrypting sensitive data"
  key_usage            = "encrypt-decrypt"
  encryption_algorithm = "aes-256-gcm"
}

# Create a signing KMS key
resource "infisical_kms_key" "signing_key" {
  project_id           = "<your-project-id>"
  name                 = "my-signing-key"
  description          = "KMS key for digital signatures"
  key_usage            = "sign-verify"
  encryption_algorithm = "RSA_4096"
}