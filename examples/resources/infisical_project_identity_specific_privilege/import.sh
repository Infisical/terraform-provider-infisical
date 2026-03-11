# Import a project identity specific privilege using its project slug, identity ID and privilege ID.
# NOTE: Import is only supported for privileges managed with the permissions_v2 block.
# Privileges using the deprecated permission block (V1) will cause a permanent plan diff after import
# because the API always returns permissions in V2 format. Migrate to permissions_v2 before importing.
terraform import infisical_project_identity_specific_privilege.example <project_slug>,<identity_id>,<privilege_id>
