#!/bin/bash
set -e  # Exit immediately if a command exits with a non-zero status

echo "Replacing files with Crossplane versions..."

DESTINATION_DIR="internal/provider/resource"
EXAMPLES_DIR="examples/resources"
SOURCE_DIR="crossplane"

if [ ! -d "$SOURCE_DIR" ]; then
  echo "Error: Source directory not found at $SOURCE_DIR"
  exit 1
fi

if [ -d "$SOURCE_DIR/project_identity_resource" ]; then
  echo "Replacing project_identity_resource"
  cp -f "$SOURCE_DIR/project_identity_resource/project_identity_resource.go" "$DESTINATION_DIR/"
  cp -f "$SOURCE_DIR/project_identity_resource/resource.tf" "$EXAMPLES_DIR/infisical_project_identity/"
fi

if [ -d "$SOURCE_DIR/project_user_resource" ]; then
  echo "Replacing project_user_resource"
  cp -f "$SOURCE_DIR/project_user_resource/project_user_resource.go" "$DESTINATION_DIR/"
  cp -f "$SOURCE_DIR/project_user_resource/resource.tf" "$EXAMPLES_DIR/infisical_project_user/"
fi

if [ -d "$SOURCE_DIR/project_group_resource" ]; then
  echo "Replacing project_group_resource"
  cp -f "$SOURCE_DIR/project_group_resource/project_group.go" "$DESTINATION_DIR/"
  cp -f "$SOURCE_DIR/project_group_resource/resource.tf" "$EXAMPLES_DIR/infisical_project_group/"
fi

if [ -d "$SOURCE_DIR/project_role_resource" ]; then
  echo "Replacing project_role_resource"
  cp -f "$SOURCE_DIR/project_role_resource/project_role_resource.go" "$DESTINATION_DIR/"
  cp -f "$SOURCE_DIR/project_role_resource/resource.tf" "$EXAMPLES_DIR/infisical_project_role/"
fi

if [ -d "$SOURCE_DIR/project_template_resource" ]; then
  echo "Replacing project_template_resource"
  cp -f "$SOURCE_DIR/project_template_resource/project_template_resource.go" "$DESTINATION_DIR/"
  cp -f "$SOURCE_DIR/project_template_resource/resource.tf" "$EXAMPLES_DIR/infisical_project_template/"
fi


for file in "$DESTINATION_DIR/secret_sync"/*.go; do

  if [ "$(basename "$file")" = "base_secret_sync.go" ]; then
    echo "Replacing base_secret_sync.go"
    mv -f "$SOURCE_DIR/secret_sync/base_secret_sync.go" "$DESTINATION_DIR/secret_sync/base_secret_sync.go"
    continue
  fi

  file_name="$(basename "$file" .go)"

  if [ ! -f "$EXAMPLES_DIR/infisical_${file_name}/resource.tf" ]; then
    echo "Error: resource.tf file not found at $EXAMPLES_DIR/infisical_${file_name}/resource.tf. Make sure you create crossplane resource examples for all secret syncs."
    exit 1
  fi

  mv "$EXAMPLES_DIR/infisical_${file_name}/resource.tf" "$EXAMPLES_DIR/infisical_${file_name}/resource.tf"

  echo "Prepared $file for Crossplane compatibility"
done



# Regenerate documentation
echo "Regenerating documentation..."
go generate ./...

echo "Resource file replacement and documentation regeneration completed successfully!"