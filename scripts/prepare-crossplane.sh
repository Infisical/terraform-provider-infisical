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

# Regenerate documentation
echo "Regenerating documentation..."
go generate ./...

echo "Resource file replacement and documentation regeneration completed successfully!"