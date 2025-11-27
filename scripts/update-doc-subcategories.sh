#!/bin/bash
set -e

# This script updates subcategories in generated Terraform documentation
# It runs after tfplugindocs to organize resources into logical groups
# for the Terraform Registry sidebar navigation.

DOCS_DIR="docs"

# Function to update subcategory in a file
update_subcategory() {
    local file="$1"
    local subcategory="$2"
    
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        sed -i '' "s/subcategory: \"\"/subcategory: \"$subcategory\"/" "$file"
    else
        # Linux
        sed -i "s/subcategory: \"\"/subcategory: \"$subcategory\"/" "$file"
    fi
}

echo "Updating documentation subcategories..."

# Resources
for file in "$DOCS_DIR/resources/"*.md; do
    [ -f "$file" ] || continue
    filename=$(basename "$file" .md)
    
    case "$filename" in
        # Secret Syncs
        secret_sync_*)
            update_subcategory "$file" "Secret Syncs";;
        
        # Secret Rotations
        secret_rotation_*)
            update_subcategory "$file" "Secret Rotations";;
        
        # Dynamic Secrets
        dynamic_secret_*)
            update_subcategory "$file" "Dynamic Secrets";;
        
        # Integrations (Deprecated)
        integration_*)
            update_subcategory "$file" "Integrations - DEPRECATED";;
        
        # App Connections
        app_connection_*)
            update_subcategory "$file" "App Connections";;
        
        # Identities
        identity|identity_*)
            update_subcategory "$file" "Identities";;
        
        # Projects
        project|project_*)
            update_subcategory "$file" "Projects";;
        
        # Approval
        access_approval_policy|secret_approval_policy)
            update_subcategory "$file" "Approval";;
        
        # Groups
        group)
            update_subcategory "$file" "Groups";;
        
        # KMS
        kms_key)
            update_subcategory "$file" "KMS";;
        
        # Secrets
        secret|secret_folder|secret_tag|secret_import)
            update_subcategory "$file" "Secrets";;
    esac
done

# Data Sources
for file in "$DOCS_DIR/data-sources/"*.md; do
    [ -f "$file" ] || continue
    filename=$(basename "$file" .md)
    
    case "$filename" in
        # Secrets
        secrets|secret_folders|secret_tag)
            update_subcategory "$file" "Secrets";;
        
        # Groups
        groups)
            update_subcategory "$file" "Groups";;
        
        # Projects
        projects)
            update_subcategory "$file" "Projects";;
        
        # Identities
        identity_details)
            update_subcategory "$file" "Identities";;
        
        # KMS
        kms_key_public_key)
            update_subcategory "$file" "KMS";;
    esac
done

# Ephemeral Resources
for file in "$DOCS_DIR/ephemeral-resources/"*.md; do
    [ -f "$file" ] || continue
    filename=$(basename "$file" .md)
    
    case "$filename" in
        # Secrets
        secret)
            update_subcategory "$file" "Secrets";;
    esac
done

echo "Subcategories updated successfully!"

