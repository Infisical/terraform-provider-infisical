---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "infisical Provider"
subcategory: ""
description: |-
  This provider allows you to interact with Infisical
---

# infisical Provider

This provider allows you to interact with Infisical

## Example Usage

```terraform
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
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `auth` (Attributes) The configuration values for authentication (see [below for nested schema](#nestedatt--auth))
- `client_id` (String, Sensitive) (DEPRECATED, Use the `auth` attribute), Machine identity client ID. Used to fetch/modify secrets for a given project.
- `client_secret` (String, Sensitive) (DEPRECATED, use `auth` attribute), Machine identity client secret. Used to fetch/modify secrets for a given project
- `host` (String) Used to point the client to fetch secrets from your self hosted instance of Infisical. If not host is provided, https://app.infisical.com is the default host. This attribute can also be set using the `INFISICAL_HOST` environment variable
- `service_token` (String, Sensitive) (DEPRECATED, Use machine identity auth), Used to fetch/modify secrets for a given project

<a id="nestedatt--auth"></a>
### Nested Schema for `auth`

Optional:

- `aws_iam` (Attributes) The configuration values for AWS IAM Auth (see [below for nested schema](#nestedatt--auth--aws_iam))
- `kubernetes` (Attributes) The configuration values for Kubernetes Auth (see [below for nested schema](#nestedatt--auth--kubernetes))
- `oidc` (Attributes) The configuration values for OIDC Auth (see [below for nested schema](#nestedatt--auth--oidc))
- `token` (String, Sensitive) The authentication token for Machine Identity Token Auth. This attribute can also be set using the `INFISICAL_TOKEN` environment variable
- `universal` (Attributes) The configuration values for Universal Auth (see [below for nested schema](#nestedatt--auth--universal))

<a id="nestedatt--auth--aws_iam"></a>
### Nested Schema for `auth.aws_iam`

Optional:

- `identity_id` (String, Sensitive) Machine identity ID. This attribute can also be set using the `INFISICAL_MACHINE_IDENTITY_ID` environment variable


<a id="nestedatt--auth--kubernetes"></a>
### Nested Schema for `auth.kubernetes`

Optional:

- `identity_id` (String, Sensitive) Machine identity ID. This attribute can also be set using the `INFISICAL_MACHINE_IDENTITY_ID` environment variable
- `service_account_token` (String, Sensitive) The service account token. This attribute can also be set using the `INFISICAL_KUBERNETES_SERVICE_ACCOUNT_TOKEN` environment variable
- `service_account_token_path` (String) The path to the service account token. This attribute can also be set using the `INFISICAL_KUBERNETES_SERVICE_ACCOUNT_TOKEN_PATH` environment variable. Default is `/var/run/secrets/kubernetes.io/serviceaccount/token`.


<a id="nestedatt--auth--oidc"></a>
### Nested Schema for `auth.oidc`

Optional:

- `identity_id` (String, Sensitive) Machine identity ID. This attribute can also be set using the `INFISICAL_MACHINE_IDENTITY_ID` environment variable
- `token_environment_variable_name` (String) The environment variable name for the OIDC JWT token. This attribute can also be set using the `INFISICAL_OIDC_AUTH_TOKEN_KEY_NAME` environment variable. Default is `INFISICAL_AUTH_JWT`.


<a id="nestedatt--auth--universal"></a>
### Nested Schema for `auth.universal`

Optional:

- `client_id` (String, Sensitive) Machine identity client ID. This attribute can also be set using the `INFISICAL_UNIVERSAL_AUTH_CLIENT_ID` environment variable
- `client_secret` (String, Sensitive) Machine identity client secret. This attribute can also be set using the `INFISICAL_UNIVERSAL_AUTH_CLIENT_SECRET` environment variable
