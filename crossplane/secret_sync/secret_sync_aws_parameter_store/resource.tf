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

resource "infisical_secret_sync_aws_parameter_store" "aws-parameter-store-secret-sync" {
  name          = "aws-parameter-store-secret-sync-demo"
  description   = "Demo of AWS Parameter Store secret sync"
  project_id    = "<project-id>"
  environment   = "<environment-slug>"
  secret_path   = "<secret-path>" # Root folder is /
  connection_id = "<app-connection-id>"

  sync_options = "{\"initial_sync_behavior\":\"overwrite-destination\",\"aws_kms_key_id\":\"<aws-kms-key-id>\",\"sync_secret_metadata_as_tags\":false,\"tags\":[{\"key\":\"tag-1\",\"value\":\"tag-1-value\"},{\"key\":\"tag-2\",\"value\":\"tag-2-value\"}]}"

  destination_config = "{\"aws_region\":\"<aws-region>\",\"path\":\"/example/secrets/\"}"
}
