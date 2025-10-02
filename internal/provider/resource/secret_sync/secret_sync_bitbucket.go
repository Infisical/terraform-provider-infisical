package resource

import (
	"context"
	"fmt"
	infisical "terraform-provider-infisical/internal/client"
	"terraform-provider-infisical/internal/pkg/terraform"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func verifyBitbucketDestinationConfigState(destinationConfig map[string]attr.Value, diags *diag.Diagnostics) bool {
	repositorySlugAttr, exists := destinationConfig["repository_slug"]
	if !exists {
		diags.AddError("Invalid destination config", "Expected 'repository_slug' to be present")
		return false
	}

	repositorySlugVal, ok := repositorySlugAttr.(types.String)
	if !ok {
		diags.AddError("Invalid destination config", "Expected 'repository_slug' to be a string type")
		return false
	}

	if repositorySlugVal.IsNull() || repositorySlugVal.IsUnknown() {
		diags.AddError("Invalid destination config", "Expected 'repository_slug' to have a value")
		return false
	}

	workspaceSlugAttr, exists := destinationConfig["workspace_slug"]
	if !exists {
		diags.AddError("Invalid destination config", "Expected 'workspace_slug' to be present")
		return false
	}

	workspaceSlugVal, ok := workspaceSlugAttr.(types.String)
	if !ok {
		diags.AddError("Invalid destination config", "Expected 'workspace_slug' to be a string type")
		return false
	}

	if workspaceSlugVal.IsNull() || workspaceSlugVal.IsUnknown() {
		diags.AddError("Invalid destination config", "Expected 'workspace_slug' to have a value")
		return false
	}

	requiredFields := []string{"repository_slug", "workspace_slug"}
	optionalFields := []string{"environment_id"}

	// Check required fields are not empty
	for _, field := range requiredFields {
		value, exists := destinationConfig[field]
		if !exists {
			diags.AddError("Invalid destination config", fmt.Sprintf("Expected '%s' to be present", field))
			return false
		}

		if terraform.IsAttrValueEmpty(value) {
			diags.AddError("Invalid destination config", fmt.Sprintf("Expected '%s' to be set", field))
			return false
		}
	}

	// Check for unexpected fields
	allowedFieldsMap := make(map[string]bool)
	for _, field := range requiredFields {
		allowedFieldsMap[field] = true
	}
	for _, field := range optionalFields {
		allowedFieldsMap[field] = true
	}

	for field := range destinationConfig {
		if !allowedFieldsMap[field] {
			if terraform.IsAttrValueEmpty(destinationConfig[field]) {
				continue
			}

			diags.AddError("Invalid destination config", fmt.Sprintf("Unexpected field '%s'. Supported destination_config fields are: %v", field, append(requiredFields, optionalFields...)))
			return false
		}
	}

	return true
}

type SecretSyncBitbucketDestinationConfigModel struct {
	RepositorySlug types.String `tfsdk:"repository_slug"`
	WorkspaceSlug  types.String `tfsdk:"workspace_slug"`
	EnvironmentId  types.String `tfsdk:"environment_id"`
}

type SecretSyncBitbucketSyncOptionsModel struct {
	InitialSyncBehavior   types.String `tfsdk:"initial_sync_behavior"`
	DisableSecretDeletion types.Bool   `tfsdk:"disable_secret_deletion"`
	KeySchema             types.String `tfsdk:"key_schema"`
}

func NewSecretSyncBitbucketResource() resource.Resource {
	return &SecretSyncBaseResource{
		App:              infisical.SecretSyncAppBitbucket,
		SyncName:         "Bitbucket",
		ResourceTypeName: "_secret_sync_bitbucket",
		AppConnection:    infisical.AppConnectionAppBitbucket,
		DestinationConfigAttributes: map[string]schema.Attribute{
			"repository_slug": schema.StringAttribute{
				Required:    true,
				Description: "The Bitbucket repository slug to sync secrets to.",
			},
			"workspace_slug": schema.StringAttribute{
				Required:    true,
				Description: "The Bitbucket workspace slug.",
			},
			"environment_id": schema.StringAttribute{
				Optional:    true,
				Description: "The Bitbucket deployment environment ID (optional).",
			},
		},
		SyncOptionsAttributes: map[string]schema.Attribute{
			"initial_sync_behavior": schema.StringAttribute{
				Required:    true,
				Description: "Specify how Infisical should resolve the initial sync to the destination. Supported options: overwrite-destination",
			},
			"disable_secret_deletion": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "When set to true, Infisical will not remove secrets from Bitbucket. Enable this option if you intend to manage some secrets manually outside of Infisical.",
				Default:     booldefault.StaticBool(false),
			},
			"key_schema": schema.StringAttribute{
				Optional:    true,
				Description: "The format to use for structuring secret keys in the Bitbucket destination.",
			},
		},

		ReadSyncOptionsForCreateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			syncOptionsMap := make(map[string]interface{})

			var syncOptions SecretSyncBitbucketSyncOptionsModel
			diags := plan.SyncOptions.As(ctx, &syncOptions, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			syncOptionsMap["initialSyncBehavior"] = syncOptions.InitialSyncBehavior.ValueString()
			syncOptionsMap["disableSecretDeletion"] = syncOptions.DisableSecretDeletion.ValueBool()
			syncOptionsMap["keySchema"] = syncOptions.KeySchema.ValueString()

			return syncOptionsMap, nil
		},

		ReadSyncOptionsFromApi: func(ctx context.Context, secretSync infisical.SecretSync) (types.Object, diag.Diagnostics) {
			syncOptionsMap := make(map[string]attr.Value)

			initialSyncBehavior, ok := secretSync.SyncOptions["initialSyncBehavior"].(string)
			if !ok {
				initialSyncBehavior = ""
			}

			disableSecretDeletion, ok := secretSync.SyncOptions["disableSecretDeletion"].(bool)
			if !ok {
				disableSecretDeletion = false
			}

			syncOptionsMap["initial_sync_behavior"] = types.StringValue(initialSyncBehavior)
			syncOptionsMap["disable_secret_deletion"] = types.BoolValue(disableSecretDeletion)

			keySchema, ok := secretSync.SyncOptions["keySchema"].(string)
			if keySchema == "" || !ok {
				syncOptionsMap["key_schema"] = types.StringNull()
			} else {
				syncOptionsMap["key_schema"] = types.StringValue(keySchema)
			}

			return types.ObjectValue(map[string]attr.Type{
				"initial_sync_behavior":   types.StringType,
				"disable_secret_deletion": types.BoolType,
				"key_schema":              types.StringType,
			}, syncOptionsMap)
		},

		ReadSyncOptionsForUpdateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel, state SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			syncOptionsMap := make(map[string]interface{})

			var syncOptions SecretSyncBitbucketSyncOptionsModel
			diags := plan.SyncOptions.As(ctx, &syncOptions, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			syncOptionsMap["initialSyncBehavior"] = syncOptions.InitialSyncBehavior.ValueString()
			syncOptionsMap["disableSecretDeletion"] = syncOptions.DisableSecretDeletion.ValueBool()
			syncOptionsMap["keySchema"] = syncOptions.KeySchema.ValueString()

			return syncOptionsMap, nil
		},

		ReadDestinationConfigForCreateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			destinationConfig := make(map[string]interface{})

			var cfg SecretSyncBitbucketDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &cfg, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			rawBitbucketCfg := map[string]attr.Value{
				"repository_slug": cfg.RepositorySlug,
				"workspace_slug":  cfg.WorkspaceSlug,
				"environment_id":  cfg.EnvironmentId,
			}

			if !verifyBitbucketDestinationConfigState(rawBitbucketCfg, &diags) {
				return nil, diags
			}

			destinationConfig["repositorySlug"] = cfg.RepositorySlug.ValueString()
			destinationConfig["workspaceSlug"] = cfg.WorkspaceSlug.ValueString()
			if !cfg.EnvironmentId.IsNull() && !cfg.EnvironmentId.IsUnknown() {
				destinationConfig["environmentId"] = cfg.EnvironmentId.ValueString()
			}

			return destinationConfig, diags
		},
		ReadDestinationConfigForUpdateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel, _ SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			destinationConfig := make(map[string]interface{})

			var cfg SecretSyncBitbucketDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &cfg, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			rawBitbucketCfg := map[string]attr.Value{
				"repository_slug": cfg.RepositorySlug,
				"workspace_slug":  cfg.WorkspaceSlug,
				"environment_id":  cfg.EnvironmentId,
			}

			if !verifyBitbucketDestinationConfigState(rawBitbucketCfg, &diags) {
				return nil, diags
			}

			destinationConfig["repositorySlug"] = cfg.RepositorySlug.ValueString()
			destinationConfig["workspaceSlug"] = cfg.WorkspaceSlug.ValueString()
			if !cfg.EnvironmentId.IsNull() && !cfg.EnvironmentId.IsUnknown() {
				destinationConfig["environmentId"] = cfg.EnvironmentId.ValueString()
			}

			return destinationConfig, diags
		},
		ReadDestinationConfigFromApi: func(ctx context.Context, secretSync infisical.SecretSync) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			repositorySlugVal, ok := secretSync.DestinationConfig["repositorySlug"].(string)
			if !ok {
				diags.AddError(
					"Invalid type",
					"Expected 'repositorySlug' to be a string but got something else",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			workspaceSlugVal, ok := secretSync.DestinationConfig["workspaceSlug"].(string)
			if !ok {
				diags.AddError(
					"Invalid type",
					"Expected 'workspaceSlug' to be a string but got something else",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			destinationConfig := map[string]attr.Value{
				"repository_slug": types.StringValue(repositorySlugVal),
				"workspace_slug":  types.StringValue(workspaceSlugVal),
				"environment_id":  types.StringNull(),
			}

			// Handle optional environment_id
			if environmentIdVal, ok := secretSync.DestinationConfig["environmentId"].(string); ok && environmentIdVal != "" {
				destinationConfig["environment_id"] = types.StringValue(environmentIdVal)
			}

			if !verifyBitbucketDestinationConfigState(destinationConfig, &diags) {
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			return types.ObjectValue(map[string]attr.Type{
				"repository_slug": types.StringType,
				"workspace_slug":  types.StringType,
				"environment_id":  types.StringType,
			}, destinationConfig)
		},
	}
}
