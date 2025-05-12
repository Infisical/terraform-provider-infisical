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

# universal authentication identity
resource "infisical_identity" "universal-auth" {
  name   = "universal-auth"
  role   = "member"
  org_id = "<org_id>"
  metadata = [
    {
      key   = "key1",
      value = "value1"
    },
    {
      key   = "key2",
      value = "value2"
    }
  ]
}

resource "infisical_identity_universal_auth" "ua-auth" {
  identity_id                 = infisical_identity.universal-auth.id
  access_token_ttl            = 2592000
  access_token_max_ttl        = 2592000 * 2
  access_token_num_uses_limit = 3
}

resource "infisical_identity_universal_auth_client_secret" "client-secret" {
  identity_id = infisical_identity.universal-auth.id

  depends_on = [infisical_identity_universal_auth.ua-auth]
}

output "client_secret" {
  sensitive = true
  value     = infisical_identity_universal_auth_client_secret.client-secret.client_secret
}

resource "infisical_identity" "aws-auth" {
  name   = "aws-auth"
  role   = "member"
  org_id = "<org_id>"
}

resource "infisical_identity_aws_auth" "aws-auth" {
  identity_id                 = infisical_identity.aws-auth.id
  access_token_ttl            = 2592000
  access_token_max_ttl        = 2592000 * 2
  access_token_num_uses_limit = 3
  allowed_principal_arns      = ["arn:aws:iam::123456789012:user/MyUserName"]
  allowed_account_ids         = ["123456789012", "123456789013"]
}

resource "infisical_identity" "azure-auth" {
  name   = "azure-auth"
  role   = "member"
  org_id = "<org_id>"
}

resource "infisical_identity_azure_auth" "azure-auth" {
  identity_id = infisical_identity.azure-auth.id
  tenant_id   = "TENANT_ID"
}

resource "infisical_identity" "gcp-auth" {
  name   = "gcp-auth"
  role   = "member"
  org_id = "<org_id>"
}

resource "infisical_identity_gcp_auth" "gcp-auth" {
  identity_id = infisical_identity.gcp-auth.id
  type        = "gce"
}

resource "infisical_identity" "k8-auth" {
  name   = "k8-auth"
  role   = "member"
  org_id = "<org_id>"
}

resource "infisical_identity_kubernetes_auth" "k8-auth" {
  identity_id        = infisical_identity.k8-auth.id
  kubernetes_host    = "http://example.com"
  token_reviewer_jwt = "ey<example>"
  allowed_namespaces = ["namespace-a", "namespace-b"]
}
