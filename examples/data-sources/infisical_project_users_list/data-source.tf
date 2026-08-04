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

# Fetch all members of a project
data "infisical_project_users_list" "all" {
  project_id = "<project-id>"
}

# Fetch specific members by username
data "infisical_project_users_list" "filtered" {
  project_id = "<project-id>"
  usernames  = ["alice@example.com", "bob@example.com"]
}

# Output all members
output "all-members" {
  value = data.infisical_project_users_list.all.members
}

# Output filtered members
output "filtered-members" {
  value = data.infisical_project_users_list.filtered.members
}
