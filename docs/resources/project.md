---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "infisical_project Resource - terraform-provider-infisical"
subcategory: ""
description: |-
  Create projects & save to Infisical. Only Machine Identity authentication is supported for this data source.
---

# infisical_project (Resource)

Create projects & save to Infisical. Only Machine Identity authentication is supported for this data source.

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

resource "infisical_project" "gcp-project" {
  name        = "GCP Project"
  slug        = "gcp-project"
  description = "This is a GCP project"
}

resource "infisical_project" "aws-project" {
  name        = "AWS Project"
  slug        = "aws-project"
  description = "This is an AWS project"
}

resource "infisical_project" "azure-project" {
  name = "Azure Project"
  slug = "azure-project"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) The name of the project
- `slug` (String) The slug of the project

### Optional

- `audit_log_retention_days` (Number) The audit log retention in days
- `description` (String) The description of the project
- `has_delete_protection` (Boolean) Whether the project has delete protection, defaults to false
- `kms_secret_manager_key_id` (String) The ID of the KMS secret manager key to use for the project
- `should_create_default_envs` (Boolean) Whether to create default environments for the project (dev, staging, prod), defaults to true
- `template_name` (String) The name of the template to use for the project

### Read-Only

- `id` (String) The ID of the project
- `last_updated` (String)
