# Infisical Terraform Provider

# Usage

```
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
  client_id     = "<>"
  client_secret = "<>"
}

data "infisical_secrets" "common-secrets" {
  env_slug     = "dev"
  workspace_id = "PROJECT_ID"
  folder_path  = "/some-folder/another-folder"
}

data "infisical_secrets" "backend-secrets" {
  env_slug     = "prod"
  workspace_id = "PROJECT_ID"
  folder_path  = "/"
}

output "all-project-secrets" {
  value = data.infisical_secrets.backend-secrets
}

output "single-secret" {
  value = data.infisical_secrets.backend-secrets.secrets["SECRET-NAME"]
}
```

# Development

Tutorials for creating Terraform providers can be found on the [HashiCorp Learn](https://learn.hashicorp.com/collections/terraform/providers-plugin-framework) platform. _Terraform Plugin Framework specific guides are titled accordingly._

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.19

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `go generate`.

## Testing

In order to run the full suite of Acceptance tests

- Set the following environment variables or save them in a .env file at the root. You can refer to the [example environment variable](./.env.test.example).
- Run the following command from root

```shell
make testacc
```
