terraform {
  required_providers {
    infisical = {
      source = "infisical.com/local/infisical"
    }
  }
}

# provider "infisical" {
#   host          = "https://app.infisical.com"
#   auth = {
#     oidc = {
#       identity_id = "<>"
#     }
#   }
# }

# resource "infisical_identity_gcp_auth" "identity-gcp-auth" {
#   identity_id = "38f6c702-3505-498d-9f8d-10534e1878ef"
# }

provider "infisical" {
  host          = "http://localhost:8088"
  client_id     = "196542ba-5a76-4d40-8cf1-666f6684bc7a"
  client_secret = "3577d59ca2557c1fc211496bbf4bc7d2d63b844a1c5ba3fd6a7a29222119a24f"
}

# resource "infisical_project_role" "billerz" {
#   project_slug = "project5-ng-hc"
#   name         = "Testerz"
#   description  = "A test rolez"
#   slug         = "testerzz"
#   permissions_v2 = jsonencode([
#       {
#         "subject": "secrets",
#         "action": [
#           "edit",
#           "read"
#         ],
#         "conditions": {
#           "secretName": {
#             "$ne": "fff",
#           },
#           "environment" = {
#             "$in": [
#               "xzsdasd",
#             ]
#           }
#         }
#         "inverted": false,
#       },
#       {
#         "subject": "secret-folders",
#         "action": [
#           "delete",
#           "edit"
#         ],
#         "inverted": true,
#       }
#     ])
# }

# resource "infisical_project_role" "billerz" {
#   project_slug = "project5-ng-hc"
#   name         = "Testerz"
#   description  = "A test rolez"
#   slug         = "testerzz"
#   # permissions = [
#   #   {
#   #     action  = "read"
#   #     subject = "secrets",
#   #   },
#   #   {
#   #     action  = "edit"
#   #     subject = "secrets",
#   #   },
#   # ]
#     permissions_v2 = [
#       {
#         "subject": "secret-folders",
#         "action": [
#           "edit",
#           "read"
#         ],
#         "inverted": true,
#         "conditions": jsonencode({
#           "environment" = {
#             "$eq": "dev"
#           }
#         })
#       },
#       {
#         "subject": "groups",
#         "action": [
#           "edit",
#           "read"
#         ],
#       },
#       {
#         "subject": "secrets",
#         "action": [
#           "edit",
#           "read"
#         ],
#         "inverted" = true
#         "conditions": jsonencode({
#           "environment" = {
#             "$eq": "dev"
#           }
#         })
#       },
#     ]
# }

# resource "infisical_project_role" "billerz" {
#   project_slug = "project5-ng-hc"
#   name         = "Testerz"
#   description  = "A test rolez"
#   slug         = "testerzz"
#   permissions_v2 = [
#       {
#         "subject": "secret-folders",
#         "action": [
#           "edit",
#           "read"
#         ],
#         "conditions": jsonencode({
#           "environment" = {
#             "$eq": "dev"
#           }
#         })
#       },
#       {
#         "subject": "secrets",
#         "action": [
#           "edit",
#         ],
#         "conditions": jsonencode({
#           "environment" = {
#             "$eq": "prod"
#           }
#         })
#       },
#       # {
#       #   "subject": "secrets",
#       #   "action": [
#       #     "edit",
#       #   ],
#       #   "conditions": jsonencode({
#       #     "environment" = {
#       #       "$eq": "prod"
#       #     }
#       #   })
#       # },
#       {
#         "subject": "secrets",
#         "action": [
#           "edit",
#           "read"
#         ],
#         "conditions": jsonencode({
#           "environment" = {
#             "$eq": "dev"
#           }
#         })
#       },
#     ]
# } 

#   name   = "machine-identity-1"
#   role   = "admin"
#   org_id = "df92581a-0fe9-42b5-b526-0a1e88ec8085"
# }


resource "infisical_identity" "machine-identity-1" {
  name   = "machine-identity-1-tf"
  role   = "admin"
  org_id = "df92581a-0fe9-42b5-b526-0a1e88ec8085"
}

# resource "infisical_identity_oidc_auth" "test" {
#   identity_id = infisical_identity.machine-identity-1.id
#   oidc_discovery_url = "https://token.actions.githubusercontent.com"
#   bound_issuer = "https://token.actions.githubusercontent.coms"
#   bound_claims = {
#     hello = "sheesh/*"
#     vals = "hello, world, zs"
#     # valsarv = "zs,hello,world"
#   }
#   bound_audiences = ["hello-world"]
#   oidc_ca_certificate = "whatz"
#   bound_subject = "repo:sheen-org/personal-assistant:ref:refs/heads/master/*"
#   access_token_ttl = 200
# }



# resource "infisical_identity_gcp_auth" "test" {
#   identity_id = infisical_identity.machine-identity-1.id
#   access_token_ttl = 200
# }

# resource "infisical_project_identity" "test-identity" {
#   project_id  = "5a6d4c12-9cd6-4ca4-ab5c-5eb84e1a77c1"
#   identity_id = infisical_identity.machine-identity-1.id
#   roles = jsonencode([
#     {
#       role_slug = "member"
#     },
#   ])

# #   roles = "[{\"role_slug\":\"member\"},{\"role_slug\":\"admin\"}]"
# }

# resource "infisical_project_identity" "test-identity" {
#   project_id  = "5a6d4c12-9cd6-4ca4-ab5c-5eb84e1a77c1"
#   identity_id = infisical_identity.machine-identity-1.id
#   roles = "   [{\"role_slug\":\"member\"},{\"role_slug\":\"admin\"}]"
# }

# resource "infisical_project_group" "group1" {
#   project_id = "09eda1f8-85a3-47a9-8a6f-e27f133b2a36"
#   group_slug = "bravolol"
#   roles = [
#    {
#     is_temporary                = true
#     role_slug                   = "viewer"
#     temporary_access_start_time = "2024-11-19T05:35:13Z"
#     temporary_range             = "1s"
#   },
#   {
#     role_slug = "no-access"
#   },
#  ]
# }

# resource "infisical_project_group" "group" {
#   project_id = "09eda1f8-85a3-47a9-8a6f-e27f133b2a36"
#   group_id = "52c70c28-9504-4b88-b5af-ca2495dd277d"
#   # group_name = "custom"
#   roles = [
#     {
#       role_slug                   = "viewer",
#       is_temporary                = false,
#     },
#     {
#       role_slug                   = "admin",
#       is_temporary                = true,
#       temporary_access_start_time = "2024-09-19T12:43:13Z",
#       temporary_range             = "1y"
#     },
#   ]
# }

# resource "infisical_project_user" "test-user" {
#   project_id = "5a6d4c12-9cd6-4ca4-ab5c-5eb84e1a77c1"
#   username   = "sheen+200@infisical.com"
#   roles = jsonencode([
#     {
#       role_slug = "admin"
#     },
#   ])
# }

# resource "infisical_secret_approval_policy" "staging" {
#     project_id = "09eda1f8-85a3-47a9-8a6f-e27f133b2a36"
#     environment_slug = "dev"
#     secret_path = "/"

#     approvers = [
#       {
#         username = "sheen+1@infisical.com"
#         type = "user"
#       },
#       {
#         type = "group"
#         id = "52c70c28-9504-4b88-b5af-ca2495dd277d"
#       },
#     ]

#     required_approvals = 2
#     enforcement_level =  "hard"
#     name = "lola"
# }

# resource "infisical_access_approval_policy" "prod" {
#     project_id = "09eda1f8-85a3-47a9-8a6f-e27f133b2a36"
#     name = "my-approval-policy-1"
#     environment_slug = "staging"
#     secret_path = "/"
#     approvers = [
#       {
#       type = "user"
#       username = "sheen+200@infisical.com"
#     },
#     {
#       type = "group"
#       id = "52c70c28-9504-4b88-b5af-ca2495dd277d"
#     },]
#     required_approvals = 1
#     enforcement_level =  "soft"
# }

# resource "infisical_access_approval_policy" "prod2" {
#     project_id = "09eda1f8-85a3-47a9-8a6f-e27f133b2a36"
#     environment_slug = "staging"
#     secret_path = "/hi"
#     approvers = ["sheen@infisical.com", "sheen+200@infisical.com"]
#     required_approvals = 1
#     enforcement_level = "soft"
# }


# HINGE HEALTH TEST :D
# data "infisical_groups" "groups" {

# }

# locals {
#   scim_groups_1 = ["asdasd"]
#   scim_groups_2 = ["custom"]
#   group_name_to_id = { for group in data.infisical_groups.groups.groups : group.name => group.id}
# }

# resource "infisical_project_group" "group" {
#   for_each = toset(local.scim_groups_1)

#   project_id = "09eda1f8-85a3-47a9-8a6f-e27f133b2a36"
#   group_id = local.group_name_to_id[each.value]
#   roles = [
#     {
#       role_slug                   = "viewer",
#       is_temporary                = false,
#     },
#     {
#       role_slug                   = "admin",
#       is_temporary                = true,
#       temporary_access_start_time = "2024-09-19T12:43:13Z",
#       temporary_range             = "1h"
#     },
#   ]
# }

# resource "infisical_project_group" "group2" {
#   for_each = toset(local.scim_groups_2)

#   project_id = "09eda1f8-85a3-47a9-8a6f-e27f133b2a36"
#   group_id = local.group_name_to_id[each.value]
#   roles = [
#     {
#       role_slug                   = "viewer",
#       is_temporary                = false,
#     },
#   ]
# }

# resource "infisical_project_group" "group3" {
#   project_id = "5a6d4c12-9cd6-4ca4-ab5c-5eb84e1a77c1"
#   group_id = "62ade129-427a-43f1-91fe-b49ac45b854e"
#   roles = jsonencode([
#     {
#       role_slug                   = "admin",
#     },
#   ])
# }

# resource "infisical_integration_gcp_secret_manager" "gcp-integration" {
#   project_id           = "09eda1f8-85a3-47a9-8a6f-e27f133b2a36"
#   service_account_json = <<-EOF
#   {
#     "type": "service_account",
#     "project_id": "daniel-test-437412",
#     "private_key_id": "4d58824fd834bbd74a1c56add8f37834745359c1",
#     "private_key": "-----BEGIN PRIVATE KEY-----\nMIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQCxsAbPMGbCIcXB\ngqPzlPw02BQ+tZXX53G7i7Jdq9UJLW4eNDDIMx/sHqaab4l2jNndEeWrVaVqZTkF\naZNQAoEw02X3GewA7uD1GdLZ5AldBZlPcpxKMo92wpXnMK6rFGvRZ+jaCOR/tm/3\nTSXLMVas8Z5v0ga39+h3lE5Yar1L6UgvxJWsZTXoon1Z7RdPPwRgV6xtI4VjZT6W\nbzbynXYqTv43NeoJUX/xZI3rDxPobznzWyVZ3tyY91263PU79l1wLLGKGC27GVhF\nw+LlfGIaWOvqz4hkI4ycYyr+XmHE5xgyX3f5nXWdclxfrtdg3q7K6xf4+JBQ8mnt\n+Gq9ZO6jAgMBAAECggEAAiGAet6q/d9FJPvxON0v7aIKAfsL5VBXTH2cDWIOe+hs\nssbUQESBfOW+zVXbWwdVQg2BOcq2rfzTkkxPweECnEdal3wWsmNeLpiwwAiV++S7\nOooUdEV4JtbIA/ZZsePtzb6d3PvIDOwItBuGRed498Jg9OWr0+N/iQ36uCUgFX3Q\nZljPtaYtH7nhovmNlvtAQG4jcbi7cfp03RV2A486L3KS765WNJUJjIe3uNBp2sVh\nqirFo+Mh51m2F0vPzEPneyNhcSOBcFJQInDEH/8Jh+vF7amXEaXWaSn0mVfluWMA\n6Ti0HO8e9FLayPPL6GhxahhUWPaX7WxA+ynwF1pkKQKBgQD0fZXF3HiDuQ/PFWdw\nYXiXVuqQAEk9g+E1pSIGiGwI2MbM2/xN3ZecV0ii1DcNhQ5cLDBKqtqTSp7TxX0B\nCYL/WG305nknhac+VGQNUpS7EWBxvp8n9ucE7zfzL9TcYvorsUTaG6qe9H3ujpC3\nsc5sPb8Zz6ECyQQgLAETTvNslQKBgQC6DWIE+RGOMTMq+zF+4rsDwcqG8LFscugm\nXUWqYddOLSEB9Lo49X1N86QDEDLFne6ytj1ELaWRWgYSrLnS8qPU4e/BI4TSR2t7\nTpuxR0L3e0nqKvS/Wl/tpn3VpNANNIVaqDrdAJ3cKT65fe7UjEUmmj2tQ5fX9xjI\ni/cbQDToVwKBgFnrAjV142DWpCjOP2/GeVp3neb+I/Ga2i4noH70l38dcugPFBjz\nIXpfY5h3IhQ31lMx8UTU13SKYiWSoWnLPMF6nV4PkYlmj17OHMoFkCvItUbAC7rg\nBJD9Bf/LnKa9RDLjjGYG/NZfJx2gkzrsCvYmM21jvlzO31SRuoeGZuKNAoGANpmw\n12bE2SblLkrzlpoxagPYTMucNghuyrt6s2rtRbsGwc0xTX/12weSbXe2fro/j+Dd\nkAGZYlO6Dob0Lc0ZeWMo+lRTKWbeSxyhomAYbgqXgYpDs1hxaIwAx88LY6SzMgzG\n4Y7JxQ+xobwsd+IGdTK0wQFiMXYJpuk0hqHMJRcCgYA7Sn1eRxlGuaXfnsqmfsad\ngnHBqA8AGjGO9A5JEw1jfme4a4oULiQ69rPzZobZbTUKkXfCljk4Yk6KkMLEVlkv\nieZ0hTmkzQBsI2/p9FME9quZM+7lb682OKMcwLL1Z4CWZWGGtk86XekKu3q0ePsI\n7vlDHDemLiQB5xO1rbQ+3A==\n-----END PRIVATE KEY-----\n",
#     "client_email": "testservice@daniel-test-437412.iam.gserviceaccount.com",
#     "client_id": "110675041026848105057",
#     "auth_uri": "https://accounts.google.com/o/oauth2/auth",
#     "token_uri": "https://oauth2.googleapis.com/token",
#     "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
#     "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/testservice%40daniel-test-437412.iam.gserviceaccount.com",
#     "universe_domain": "googleapis.com"
#   }
#   EOF

#   gcp_project_id       = "daniel-test-437412"
#   environment          = "staging"
#   secret_path          = "/"
#   options = {
#     secret_prefix = "xdzz"
#   }
# }


# resource "infisical_integration_aws_parameter_store" "parameter-store-integration" {
#   project_id  = "09eda1f8-85a3-47a9-8a6f-e27f133b2a36"
#   environment = "dev"
#   secret_path          = "/dynamic-secret-test"
#   parameter_store_path = "/example/secrets"
#   aws_region        = "us-east-2" // example, us-east-2
#   assume_role_arn = "arn:aws:iam::123123:role/infisical-sm"
#   options = {
#     should_disable_delete = false,
#     aws_tags = [
#       {
#         key   = "key",
#         value = "value"
#       },
#     ]
#   }
# }

# resource "infisical_integration_databricks" "db-integration" {
#   project_id  = "09eda1f8-85a3-47a9-8a6f-e27f133b2a36"
#   environment = "dev"
#   databricks_host         = "https://afc-2a42f142-bb11.cloud.databricks.com"
#   databricks_token        = "booyazz"
#   databricks_secret_scope = "prod"
#   secret_path = "/dynamic-secret-test"
# }

# resource "infisical_integration_aws_secrets_manager" "secrets-manager-integration" {
#   project_id  = "09eda1f8-85a3-47a9-8a6f-e27f133b2a36"
#   aws_region  = "us-east-2" // example, us-east-2
#   environment = "prod"   // example, dev
#   secret_path = "/" // example, /folder, or /
#   secrets_manager_path = "/example/secre" # Only required if mapping_behavior is one-to-one
#   access_key_id     = "zzz"
#   secret_access_key = "fffzz"
#   # assume_role_arn = "arn:aws:iam::<aws-account  -id>:role/<role-name>"
#   # options = {
#   #   metadata_sync_mode="custom"
#   # }
# }

# resource "infisical_integration_aws_secrets_manager" "secrets-manager-integration-1" {
#   project_id  = "09eda1f8-85a3-47a9-8a6f-e27f133b2a36"
#   mapping_behavior = "one-to-one"
#   aws_region  = "us-east-2" // example, us-east-2
#   environment = "prod"   // example, dev
#   secret_path = "/" // example, /folder, or /
#   access_key_id     = "zzz"
#   secret_access_key = "fff"
#   # assume_role_arn = "arn:aws:iam::<aws-account-id>:role/<role-name>"
#   options = {
#     metadata_sync_mode="secret-metadata"
#   }
# }

# resource "infisical_secret_folder" "test_folder" {
#   environment_slug = "dev"
#   folder_path = "/"
#   name = "tester-folder"
#   project_id = "09eda1f8-85a3-47a9-8a6f-e27f133b2a36"
# }

# resource "infisical_secret_folder" "test_folder1" {
#   environment_slug = "dev"
#   folder_path = "/tester-folder"
#   name = "deep12"
#   project_id = "09eda1f8-85a3-47a9-8a6f-e27f133b2a36"
# }

# resource "infisical_secret_folder" "test_folder2" {
#   environment_slug = "dev"
#   folder_path = "/tester-folder/deep12"
#   name = "omg"
#   project_id = "09eda1f8-85a3-47a9-8a6f-e27f133b2a36"
# }

# certificate-test
# resource "infisical_project_identity_specific_privilege" "test-privilege" {
#   project_slug = "certificate-test-ju-ej"
#   identity_id  = "a4b45a49-bd0c-451e-b440-23aee06f01df"
#   slug = "hays"
#   permissions_v2 = [
#     {
#       action = ["read", "edit"]
#       subject = "secret-folders",
#     },
#     {
#       action = ["read", "edit"]
#       subject = "secrets"
#       conditions = jsonencode({
#         secretPath  = {
#           "$eq" = "/"
#         }
#         environment = {
#           "$in" = ["dev", "prod"]
#           "$eq" = "dev"
#         }
#       })
#     },
#   ]
# }

resource "infisical_app_connection_gcp" "app-connection-gcp" {
  name = "gcp-app-connects"
  method = "service-account-impersonation"
  credentials = {
    service_account_email = "service-account-df92581a-0fe9@my-duplicate-project.iam.gserviceaccount.com"
  }
  description = "I am a test app connections"
}

# resource "infisical_secret_sync_gcp_secret_manager" "secret_manager" {
#   name = "gcp-sync"
#   environment = "dev"
#   connection_id = infisical_app_connection_gcp.app-connection-gcp.id
#   secret_path = "/"
#   gcp_project_id = "my-duplicate-project"
#   project_id = "f4517f4c-8b61-4727-8aef-5ae2807126fb"
#   initial_sync_behavior = "import-prioritize-destination"
#   description = "I am a test secret sync"
# }

resource "infisical_secret_sync_gcp_secret_manager" "secret_manager_test" {
  name = "gcp-sync-tests"
  description = "I am a test secret sync"
  project_id = "f4517f4c-8b61-4727-8aef-5ae2807126fb"
  environment = "prod"
  secret_path = "/"
  connection_id = infisical_app_connection_gcp.app-connection-gcp.id

  sync_options = {
    initial_sync_behavior = "import-prioritize-destination"
  }
  destination_config = {
     project_id = "my-duplicate-project"
  }
}


# resource "infisical_secret_sync_azure_key_vault" "app-configuration-demo" {
#   name          = "demo-sync"
#   description   = "This is a demo sync."
#   project_id    = "f4517f4c-8b61-4727-8aef-5ae2807126fb"
#   environment   = "dev"
#   secret_path   = "/"
#   connection_id = "0c421791-f61e-498e-b2ea-e8abb030c4a8" # The ID of your Azure App Connection

#   sync_options = {
#     initial_sync_behavior = "overwrite-destination"
#   }
#   destination_config = {
#     vault_base_url = "a", # https://example.vault.azure.net/
#   }
# }

# resource "infisical_secret_sync_azure_app_configuration" "app-configuration-demo" {
#   name          = "demo-sync"
#   description   = "This is a demo sync"
#   project_id    = "f4517f4c-8b61-4727-8aef-5ae2807126fb"
#   environment   = "dev"
#   secret_path   = "/"
#   connection_id = "0c421791-f61e-498e-b2ea-e8abb030c4a8" # The ID of your Azure App Connection
#   sync_options = {
#     initial_sync_behavior = "overwrite-destination"
#   }
#   destination_config = {
#     configuration_url = "https://infisical-configuration-integration-test.azconfig.io", # https://example.azconfig.io
#     label = "CHUY"
#   }
# }

# resource "infisical_project" "gcp-projectz" {
#   description = "helloz"
#   name = "project5"
#   slug = "project5-ng-hcz"
#   template_name = "default"
# }


# resource "infisical_secret_sync_aws_parameter_store" "aws-parameter-store-secret-sync" {
#   name          = "lols"
#   description   = "Demo of AWS Parameter Store secret sync"
#   project_id    = "f4517f4c-8b61-4727-8aef-5ae2807126fb"
#   environment   = "dev"
#   secret_path   = "/" # Root folder is /
#   connection_id = "53ed362c-5d5d-4f76-afdb-6c27ba85c3e5"

#   sync_options = {
#     initial_sync_behavior        = "overwrite-destination", # Supported options: overwrite-destination, import-prioritize-source, import-prioritize-destination
#     # sync_secret_metadata_as_tags = false,
#     # tags = [
#     #   {
#     #     key   = "tag-2"
#     #     value = "tag-2-value"
#     #   },
#     # ]
#   }

#   destination_config = {
#     aws_region = "us-east-1" # E.g us-east-1
#     path       = "/example/secrets/"
#   }
# }

# resource "infisical_secret_sync_aws_secrets_manager" "aws-secrets-manager-secret-sync" {
#   name          = "aws-secrets"
#   description   = "Demo"
#   project_id    = "f4517f4c-8b61-4727-8aef-5ae2807126fb"
#   environment   = "staging"
#   secret_path   = "/" # Root folder is /
#   connection_id = "53ed362c-5d5d-4f76-afdb-6c27ba85c3e5"

#   sync_options = {
#     initial_sync_behavior        = "import-prioritize-source", # Supported options: overwrite-destination, import-prioritize-source, import-prioritize-destination
#     # aws_kms_key_id               = "<aws-kms-key-id>",
#     sync_secret_metadata_as_tags = true,
#     tags = [
#       {
#         key   = "tag-1"
#         value = "tag-1-valuezs"
#       },
#       {
#         key   = "tag-2"
#         value = "tag-2-valuez"
#       },
#     ]
#   }

#   destination_config = {
#     aws_region                      = "us-east-1"      # E.g us-east-1
#     mapping_behavior                = "one-to-one"       # Supported options: many-to-one, one-to-one
#     # aws_secrets_manager_secret_name = "lols" # Only required when mapping behavior is 'many-to-one'
#   }
# }

resource "infisical_project" "aws-project" {
  name        = "AWS Project"
  slug        = "aws-project"
  description = "This is an AWS project"
}
