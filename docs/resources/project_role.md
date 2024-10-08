---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "infisical_project_role Resource - terraform-provider-infisical"
subcategory: ""
description: |-
  Create custom project roles & save to Infisical. Only Machine Identity authentication is supported for this data source.
---

# infisical_project_role (Resource)

Create custom project roles & save to Infisical. Only Machine Identity authentication is supported for this data source.

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
  host          = "https://app.infisical.com" # Only required if using self hosted instance of Infisical, default is https://app.infisical.com
  client_id     = "<>"
  client_secret = "<>"
}

resource "infisical_project" "example" {
  name = "example"
  slug = "example"
}

resource "infisical_project_role" "biller" {
  project_slug = infisical_project.example.slug
  name         = "Tester"
  description  = "A test role"
  slug         = "tester"
  permissions = [
    {
      action  = "read"
      subject = "secrets",
      conditions = {
        environment = "dev"
        secret_path = "/dev"
      }
    },
  ]
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) The name for the new role
- `permissions` (Attributes List) The permissions assigned to the project role (see [below for nested schema](#nestedatt--permissions))
- `project_slug` (String) The slug of the project to create role
- `slug` (String) The slug for the new role

### Optional

- `description` (String) The description for the new role. Defaults to an empty string.

### Read-Only

- `id` (String) The ID of the role

<a id="nestedatt--permissions"></a>
### Nested Schema for `permissions`

Required:

- `action` (String) Describe what action an entity can take. Enum: create,edit,delete,read
- `subject` (String) Describe what action an entity can take. Enum: role,member,groups,settings,integrations,webhooks,service-tokens,environments,tags,audit-logs,ip-allowlist,workspace,secrets,secret-rollback,secret-approval,secret-rotation,identity,certificate-authorities,certificates,certificate-templates,kms,pki-alerts,pki-collections

Optional:

- `conditions` (Attributes) The conditions to scope permissions (see [below for nested schema](#nestedatt--permissions--conditions))

<a id="nestedatt--permissions--conditions"></a>
### Nested Schema for `permissions.conditions`

Optional:

- `environment` (String) The environment slug this permission should allow.
- `secret_path` (String) The secret path this permission should be scoped to
