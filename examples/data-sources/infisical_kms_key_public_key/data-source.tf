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

# Get public key information for a signing KMS key
data "infisical_kms_key_public_key" "example" {
  key_id = "<your-signing-kms-key-id>"
}

# Output the public key
output "public_key" {
  value       = data.infisical_kms_key_public_key.example.public_key
  description = "The public key for cryptographic operations"
}

# Output available signing algorithms
output "signing_algorithms" {
  value       = data.infisical_kms_key_public_key.example.signing_algorithms
  description = "Available signing algorithms for this key"
}