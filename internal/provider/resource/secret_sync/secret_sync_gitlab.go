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

type GitlabSyncScope string

const (
	GitlabSyncScopeProject GitlabSyncScope = "project"
	GitlabSyncScopeGroup   GitlabSyncScope = "group"
)

func verifyGitlabDestinationConfigState(destinationConfig map[string]attr.Value, diags *diag.Diagnostics) bool {
	scopeAttr, exists := destinationConfig["scope"]
	if !exists {
		diags.AddError("Invalid destination config", "Expected 'scope' to be present")
		return false
	}

	scopeVal, ok := scopeAttr.(types.String)
	if !ok {
		diags.AddError("Invalid destination config", "Expected 'scope' to be a string type")
		return false
	}

	if scopeVal.IsNull() || scopeVal.IsUnknown() {
		diags.AddError("Invalid destination config", "Expected 'scope' to have a value")
		return false
	}

	scope := GitlabSyncScope(scopeVal.ValueString())
	requiredFields := []string{"scope", "target_environment"}
	optionalFields := []string{"should_protect_secrets", "should_mask_secrets", "should_hide_secrets"}

	switch scope {
	case GitlabSyncScopeProject:
		requiredFields = append(requiredFields, "project_id")
		optionalFields = append(optionalFields, "project_name")

	case GitlabSyncScopeGroup:
		requiredFields = append(requiredFields, "group_id")
		optionalFields = append(optionalFields, "group_name")

	default:
		diags.AddError("Invalid destination config", fmt.Sprintf("Invalid scope '%s'. Must be 'project' or 'group'", scope))
		return false
	}

	// Check required fields are not empty
	for _, field := range requiredFields {
		value, exists := destinationConfig[field]
		if !exists {
			diags.AddError("Invalid destination config", fmt.Sprintf("Expected '%s' to be present when scope is '%s'", field, scope))
			return false
		}

		// Check if the value is null, unknown, or empty based on its type
		if terraform.IsAttrValueEmpty(value) {
			diags.AddError("Invalid destination config", fmt.Sprintf("Expected '%s' to be set when scope is '%s'", field, scope))
			return false
		}
	}

	// Build allowed fields map
	allowedFieldsMap := make(map[string]bool)
	for _, field := range requiredFields {
		allowedFieldsMap[field] = true
	}
	for _, field := range optionalFields {
		allowedFieldsMap[field] = true
	}

	// Check for unexpected fields
	for field := range destinationConfig {
		if !allowedFieldsMap[field] {
			if terraform.IsAttrValueEmpty(destinationConfig[field]) {
				continue
			}

			diags.AddError("Invalid destination config", fmt.Sprintf("Unexpected field '%s' for scope '%s'", field, scope))
			return false
		}
	}

	return true
}

type SecretSyncGitlabDestinationConfigModel struct {
	Scope                types.String `tfsdk:"scope"`
	ProjectId            types.String `tfsdk:"project_id"`
	ProjectName          types.String `tfsdk:"project_name"`
	GroupId              types.String `tfsdk:"group_id"`
	GroupName            types.String `tfsdk:"group_name"`
	TargetEnvironment    types.String `tfsdk:"target_environment"`
	ShouldProtectSecrets types.Bool   `tfsdk:"should_protect_secrets"`
	ShouldMaskSecrets    types.Bool   `tfsdk:"should_mask_secrets"`
	ShouldHideSecrets    types.Bool   `tfsdk:"should_hide_secrets"`
}

type SecretSyncGitlabSyncOptionsModel struct {
	InitialSyncBehavior   types.String `tfsdk:"initial_sync_behavior"`
	DisableSecretDeletion types.Bool   `tfsdk:"disable_secret_deletion"`
	KeySchema             types.String `tfsdk:"key_schema"`
}

func NewSecretSyncGitlabResource() resource.Resource {
	return &SecretSyncBaseResource{
		App:              infisical.SecretSyncAppGitlab,
		SyncName:         "GitLab",
		ResourceTypeName: "_secret_sync_gitlab",
		AppConnection:    infisical.AppConnectionAppGitlab,
		DestinationConfigAttributes: map[string]schema.Attribute{
			"scope": schema.StringAttribute{
				Required:    true,
				Description: "The GitLab scope that secrets should be synced to. Supported options: 'project', 'group'",
			},
			"project_id": schema.StringAttribute{
				Optional:    true,
				Description: "The GitLab Project ID to sync secrets to. Required when scope is 'project'.",
			},
			"project_name": schema.StringAttribute{
				Optional:    true,
				Description: "The GitLab Project Name to sync secrets to. Optional when scope is 'project'.",
			},
			"group_id": schema.StringAttribute{
				Optional:    true,
				Description: "The GitLab Group ID to sync secrets to. Required when scope is 'group'.",
			},
			"group_name": schema.StringAttribute{
				Optional:    true,
				Description: "The GitLab Group Name to sync secrets to. Optional when scope is 'group'.",
			},
			"target_environment": schema.StringAttribute{
				Required:    true,
				Description: "The GitLab environment scope that secrets should be synced to. (default: *)",
			},
			"should_protect_secrets": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether variables should be protected",
				Default:     booldefault.StaticBool(false),
			},
			"should_mask_secrets": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether variables should be masked in logs",
				Default:     booldefault.StaticBool(true),
			},
			"should_hide_secrets": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether variables should be hidden",
				Default:     booldefault.StaticBool(false),
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
				Description: "When set to true, Infisical will not remove secrets from GitLab. Enable this option if you intend to manage some secrets manually outside of Infisical.",
				Default:     booldefault.StaticBool(false),
			},
			"key_schema": schema.StringAttribute{
				Optional:    true,
				Description: "The format to use for structuring secret keys in the GitLab destination.",
			},
		},

		ReadSyncOptionsForCreateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]any, diag.Diagnostics) {
			syncOptionsMap := make(map[string]any)

			var syncOptions SecretSyncGitlabSyncOptionsModel
			diags := plan.SyncOptions.As(ctx, &syncOptions, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			syncOptionsMap["initialSyncBehavior"] = syncOptions.InitialSyncBehavior.ValueString()
			syncOptionsMap["disableSecretDeletion"] = syncOptions.DisableSecretDeletion.ValueBool()

			if !syncOptions.KeySchema.IsNull() && syncOptions.KeySchema.ValueString() != "" {
				syncOptionsMap["keySchema"] = syncOptions.KeySchema.ValueString()
			}

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

		ReadSyncOptionsForUpdateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel, _ SecretSyncBaseResourceModel) (map[string]any, diag.Diagnostics) {
			syncOptionsMap := make(map[string]any)

			var syncOptions SecretSyncGitlabSyncOptionsModel
			diags := plan.SyncOptions.As(ctx, &syncOptions, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			syncOptionsMap["initialSyncBehavior"] = syncOptions.InitialSyncBehavior.ValueString()
			syncOptionsMap["disableSecretDeletion"] = syncOptions.DisableSecretDeletion.ValueBool()

			if !syncOptions.KeySchema.IsNull() && syncOptions.KeySchema.ValueString() != "" {
				syncOptionsMap["keySchema"] = syncOptions.KeySchema.ValueString()
			}

			return syncOptionsMap, nil
		},

		ReadDestinationConfigForCreateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]any, diag.Diagnostics) {
			destinationConfig := make(map[string]any)

			var gitlabCfg SecretSyncGitlabDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &gitlabCfg, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			// Convert to map for validation
			destinationConfigAttrMap := map[string]attr.Value{
				"scope":                  gitlabCfg.Scope,
				"project_id":             gitlabCfg.ProjectId,
				"project_name":           gitlabCfg.ProjectName,
				"group_id":               gitlabCfg.GroupId,
				"group_name":             gitlabCfg.GroupName,
				"target_environment":     gitlabCfg.TargetEnvironment,
				"should_protect_secrets": gitlabCfg.ShouldProtectSecrets,
				"should_mask_secrets":    gitlabCfg.ShouldMaskSecrets,
				"should_hide_secrets":    gitlabCfg.ShouldHideSecrets,
			}

			// Validate configuration
			if !verifyGitlabDestinationConfigState(destinationConfigAttrMap, &diags) {
				return nil, diags
			}

			// Build config based on scope
			scope := GitlabSyncScope(gitlabCfg.Scope.ValueString())
			destinationConfig["scope"] = string(scope)
			destinationConfig["targetEnvironment"] = gitlabCfg.TargetEnvironment.ValueString()

			switch scope {
			case GitlabSyncScopeProject:
				destinationConfig["projectId"] = gitlabCfg.ProjectId.ValueString()
				if !gitlabCfg.ProjectName.IsNull() && gitlabCfg.ProjectName.ValueString() != "" {
					destinationConfig["projectName"] = gitlabCfg.ProjectName.ValueString()
				}

			case GitlabSyncScopeGroup:
				destinationConfig["groupId"] = gitlabCfg.GroupId.ValueString()
				if !gitlabCfg.GroupName.IsNull() && gitlabCfg.GroupName.ValueString() != "" {
					destinationConfig["groupName"] = gitlabCfg.GroupName.ValueString()
				}
			}

			// Add boolean flags
			if !gitlabCfg.ShouldProtectSecrets.IsNull() {
				destinationConfig["shouldProtectSecrets"] = gitlabCfg.ShouldProtectSecrets.ValueBool()
			}

			if !gitlabCfg.ShouldMaskSecrets.IsNull() {
				destinationConfig["shouldMaskSecrets"] = gitlabCfg.ShouldMaskSecrets.ValueBool()
			}

			if !gitlabCfg.ShouldHideSecrets.IsNull() {
				destinationConfig["shouldHideSecrets"] = gitlabCfg.ShouldHideSecrets.ValueBool()
			}

			return destinationConfig, diags
		},

		ReadDestinationConfigForUpdateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel, _ SecretSyncBaseResourceModel) (map[string]any, diag.Diagnostics) {
			destinationConfig := make(map[string]any)

			var gitlabCfg SecretSyncGitlabDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &gitlabCfg, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			// Convert to map for validation
			destinationConfigAttrMap := map[string]attr.Value{
				"scope":                  gitlabCfg.Scope,
				"project_id":             gitlabCfg.ProjectId,
				"project_name":           gitlabCfg.ProjectName,
				"group_id":               gitlabCfg.GroupId,
				"group_name":             gitlabCfg.GroupName,
				"target_environment":     gitlabCfg.TargetEnvironment,
				"should_protect_secrets": gitlabCfg.ShouldProtectSecrets,
				"should_mask_secrets":    gitlabCfg.ShouldMaskSecrets,
				"should_hide_secrets":    gitlabCfg.ShouldHideSecrets,
			}

			// Validate configuration
			if !verifyGitlabDestinationConfigState(destinationConfigAttrMap, &diags) {
				return nil, diags
			}

			// Build config based on scope
			scope := GitlabSyncScope(gitlabCfg.Scope.ValueString())
			destinationConfig["scope"] = string(scope)
			destinationConfig["targetEnvironment"] = gitlabCfg.TargetEnvironment.ValueString()

			switch scope {
			case GitlabSyncScopeProject:
				destinationConfig["projectId"] = gitlabCfg.ProjectId.ValueString()
				if !gitlabCfg.ProjectName.IsNull() && gitlabCfg.ProjectName.ValueString() != "" {
					destinationConfig["projectName"] = gitlabCfg.ProjectName.ValueString()
				}

			case GitlabSyncScopeGroup:
				destinationConfig["groupId"] = gitlabCfg.GroupId.ValueString()
				if !gitlabCfg.GroupName.IsNull() && gitlabCfg.GroupName.ValueString() != "" {
					destinationConfig["groupName"] = gitlabCfg.GroupName.ValueString()
				}
			}

			if !gitlabCfg.ShouldProtectSecrets.IsNull() {
				destinationConfig["shouldProtectSecrets"] = gitlabCfg.ShouldProtectSecrets.ValueBool()
			}

			if !gitlabCfg.ShouldMaskSecrets.IsNull() {
				destinationConfig["shouldMaskSecrets"] = gitlabCfg.ShouldMaskSecrets.ValueBool()
			}

			if !gitlabCfg.ShouldHideSecrets.IsNull() {
				destinationConfig["shouldHideSecrets"] = gitlabCfg.ShouldHideSecrets.ValueBool()
			}

			return destinationConfig, diags
		},

		ReadDestinationConfigFromApi: func(ctx context.Context, secretSync infisical.SecretSync) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			scopeVal, ok := secretSync.DestinationConfig["scope"].(string)
			if !ok {
				diags.AddError(
					"Invalid scope type",
					"Expected 'scope' to be a string but got something else",
				)
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			destinationConfig := map[string]attr.Value{
				"scope": types.StringValue(scopeVal),
			}

			// Read project ID
			if projectIdVal, ok := secretSync.DestinationConfig["projectId"].(string); ok {
				destinationConfig["project_id"] = types.StringValue(projectIdVal)
			} else {
				destinationConfig["project_id"] = types.StringNull()
			}

			// Read project name (optional)
			if projectNameVal, ok := secretSync.DestinationConfig["projectName"].(string); ok && projectNameVal != "" {
				destinationConfig["project_name"] = types.StringValue(projectNameVal)
			} else {
				destinationConfig["project_name"] = types.StringNull()
			}

			// Read group ID
			if groupIdVal, ok := secretSync.DestinationConfig["groupId"].(string); ok {
				destinationConfig["group_id"] = types.StringValue(groupIdVal)
			} else {
				destinationConfig["group_id"] = types.StringNull()
			}

			// Read group name (optional)
			if groupNameVal, ok := secretSync.DestinationConfig["groupName"].(string); ok && groupNameVal != "" {
				destinationConfig["group_name"] = types.StringValue(groupNameVal)
			} else {
				destinationConfig["group_name"] = types.StringNull()
			}

			// Read target environment
			if targetEnvVal, ok := secretSync.DestinationConfig["targetEnvironment"].(string); ok {
				destinationConfig["target_environment"] = types.StringValue(targetEnvVal)
			} else {
				destinationConfig["target_environment"] = types.StringNull()
			}

			// Read boolean flags with defaults
			if shouldProtectVal, ok := secretSync.DestinationConfig["shouldProtectSecrets"].(bool); ok {
				destinationConfig["should_protect_secrets"] = types.BoolValue(shouldProtectVal)
			} else {
				destinationConfig["should_protect_secrets"] = types.BoolValue(false)
			}

			if shouldMaskVal, ok := secretSync.DestinationConfig["shouldMaskSecrets"].(bool); ok {
				destinationConfig["should_mask_secrets"] = types.BoolValue(shouldMaskVal)
			} else {
				destinationConfig["should_mask_secrets"] = types.BoolValue(true)
			}

			if shouldHideVal, ok := secretSync.DestinationConfig["shouldHideSecrets"].(bool); ok {
				destinationConfig["should_hide_secrets"] = types.BoolValue(shouldHideVal)
			} else {
				destinationConfig["should_hide_secrets"] = types.BoolValue(false)
			}

			return types.ObjectValue(map[string]attr.Type{
				"scope":                  types.StringType,
				"project_id":             types.StringType,
				"project_name":           types.StringType,
				"group_id":               types.StringType,
				"group_name":             types.StringType,
				"target_environment":     types.StringType,
				"should_protect_secrets": types.BoolType,
				"should_mask_secrets":    types.BoolType,
				"should_hide_secrets":    types.BoolType,
			}, destinationConfig)
		},
	}
}
