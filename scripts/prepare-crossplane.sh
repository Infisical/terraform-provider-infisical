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



# In the SOURCE_DIR/secret_sync folder, it should recursively go over each file except the base_secret_sync.go file, and replace `CrossplaneCompatible: false` with `CrossplaneCompatible: true`
for file in "$DESTINATION_DIR/secret_sync"/*.go; do
  if [ "$(basename "$file")" != "base_secret_sync.go" ]; then
    sed -i.bak 's/CrossplaneCompatible: false/CrossplaneCompatible: true/g' "$file"
    rm -f "${file}.bak"

    # Get the file name without the extension
    file_name="$(basename "$file" .go)"
    # In the EXAMPLES_DIR/infisical_${file_name} folder, remove the resource.tf file, and rename the crossplane_resource to resource.tf
    mv "$EXAMPLES_DIR/infisical_${file_name}/crossplane_resource.tf" "$EXAMPLES_DIR/infisical_${file_name}/resource.tf"

    echo "Prepared $file for Crossplane compatibility"
  fi
done

# Regenerate documentation
echo "Regenerating documentation..."
go generate ./...

echo "Resource file replacement and documentation regeneration completed successfully!"