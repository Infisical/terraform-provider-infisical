terraform {
  required_providers {
    infisical = {
      # version = <latest version>
      source = "hashicorp.com/edu/infisical"
    }
  }
}

provider "infisical" {
  host          = "http://localhost:8080" # Only required if using self hosted instance of Infisical, default is https://app.infisical.com
  client_id     = "72c7a82f-051f-4f72-9d42-57f3401fffdd"
  client_secret = "91db2a49e38797d444fa8b5ba4d164c9ec0732bdc1ee9990673ea8fe09908310"
}

# universal authentication identity
resource "infisical_identity" "universal-auth" {
  name   = "universal-auth"
  role   = "member"
  org_id = "180870b7-f464-4740-8ffe-9d11c9245ea7"
}

output "universal_auth_identity_id" {
  value = infisical_identity.universal-auth.id
}


resource "infisical_project_identity" "test-identity" {
  project_id  = "0a7179b4-931b-4e94-948a-b37b38f17a35"
  identity_id = infisical_identity.universal-auth.id
  roles = [
    {
      role_slug = "admin"
    }
  ]
}

# resource "infisical_identity_universal_auth" "ua-auth" {
#   identity_id                 = infisical_identity.universal-auth.id
#   access_token_ttl            = 2592000
#   access_token_max_ttl        = 2592000 * 2
#   access_token_num_uses_limit = 3
# }

# resource "infisical_identity_universal_auth_client_secret" "client-secret" {
#   identity_id = infisical_identity.universal-auth.id

#   depends_on = [infisical_identity_universal_auth.ua-auth]
# }

# output "client_secret" {
#   sensitive = true
#   value     = infisical_identity_universal_auth_client_secret.client-secret.client_secret
# }

# resource "infisical_identity" "aws-auth" {
#   name   = "aws-auth"
#   role   = "member"
#   org_id = "<org_id>"
# }

# resource "infisical_identity_aws_auth" "aws-auth" {
#   identity_id                 = infisical_identity.aws-auth.id
#   access_token_ttl            = 2592000
#   access_token_max_ttl        = 2592000 * 2
#   access_token_num_uses_limit = 3
#   allowed_principal_arns      = ["arn:aws:iam::123456789012:user/MyUserName"]
#   allowed_account_ids         = ["123456789012", "123456789013"]
# }

# resource "infisical_identity" "azure-auth" {
#   name   = "azure-auth"
#   role   = "member"
#   org_id = "<org_id>"
# }

# resource "infisical_identity_azure_auth" "azure-auth" {
#   identity_id = infisical_identity.azure-auth.id
#   tenant_id   = "TENANT_ID"
# }

# resource "infisical_identity" "gcp-auth" {
#   name   = "gcp-auth"
#   role   = "member"
#   org_id = "<org_id>"
# }

# resource "infisical_identity_gcp_auth" "gcp-auth" {
#   identity_id = infisical_identity.gcp-auth.id
#   type        = "gce"
# }

# resource "infisical_identity" "k8-auth" {
#   name   = "k8-auth"
#   role   = "member"
#   org_id = "<org_id>"
# }

# resource "infisical_identity_kubernetes_auth" "k8-auth" {
#   identity_id        = infisical_identity.k8-auth.id
#   kubernetes_host    = "http://example.com"
#   token_reviewer_jwt = "ey<example>"
#   allowed_namespaces = ["namespace-a", "namespace-b"]
# }

