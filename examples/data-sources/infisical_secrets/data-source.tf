data "infisical_secrets" "my-secrets" {}

output "all-project-secrets" {
  value = data.infisical_secrets.my-secrets.secrets
}


output "single-secret" {
  value = data.infisical_secrets.my-secrets.secrets["NAME-OF-SECRET"]
}
