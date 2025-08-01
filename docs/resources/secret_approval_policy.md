---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "infisical_secret_approval_policy Resource - terraform-provider-infisical"
subcategory: ""
description: |-
  Create secret approval policy for your projects
---

# infisical_secret_approval_policy (Resource)

Create secret approval policy for your projects

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

resource "infisical_project" "example" {
  name = "example"
  slug = "example"
}

resource "infisical_secret_approval_policy" "prod-policy" {
  project_id        = infisical_project.example.id
  name              = "my-prod-policy"
  environment_slugs = ["prod"]
  secret_path       = "/"
  approvers = [
    {
      type = "group"
      id   = "52c70c28-9504-4b88-b5af-ca2495dd277d"
    },
    {
      type     = "user"
      username = "name@infisical.com"
  }]
  required_approvals = 1
  enforcement_level  = "hard"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `approvers` (Attributes Set) The required approvers (see [below for nested schema](#nestedatt--approvers))
- `project_id` (String) The ID of the project to add the secret approval policy
- `required_approvals` (Number) The number of required approvers
- `secret_path` (String) The secret path to apply the secret approval policy to

### Optional

- `allow_self_approval` (Boolean) Whether to allow the  approvers to approve their own changes
- `enforcement_level` (String) The enforcement level of the policy. This can either be hard or soft
- `environment_slug` (String) (DEPRECATED, Use environment_slugs instead) The environment to apply the secret approval policy to
- `environment_slugs` (List of String) The environments to apply the secret approval policy to
- `name` (String) The name of the secret approval policy

### Read-Only

- `id` (String) The ID of the secret approval policy

<a id="nestedatt--approvers"></a>
### Nested Schema for `approvers`

Required:

- `type` (String) The type of approver. Either group or user

Optional:

- `id` (String) The ID of the approver
- `username` (String) The username of the approver. By default, this is the email
