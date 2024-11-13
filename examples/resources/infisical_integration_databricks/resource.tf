terraform {
  required_providers {
    infisical = {
      # version = <latest version>
      source = "infisical/infisical"
    }
  }
}

provider "infisical" {
  host          = "https://app.infisical.com" # Only required if using self hosted instance of Infisical, default is https://app.infisical.com
  client_id     = "<machine-identity-client-id>"
  client_secret = "<machine-identity-client-secret>"
}

resource "infisical_integration_databricks" "db-integration" {
  project_id  = "<project-id>"
  environment = "<env-slug>"

  databricks_host         = "<databricks-host>" # Example: https://afc-2a42f142-bb11.cloud.databricks.com
  databricks_token        = "<databricks-personal-access-token>"
  databricks_secret_scope = "<databricks-secret-scope>"

  secret_path = "/some/infisical/folder" # "/" is the root folder 

}