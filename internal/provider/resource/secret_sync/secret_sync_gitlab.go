package resource

import (
	"context"
	infisical "terraform-provider-infisical/internal/client"
	infisicalclient "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type SecretSyncGitlabDestinationConfigModel struct {
	Scope                types.String `tfsdk:"scope"`
	ProjectId            types.String `tfsdk:"project_id"`
	ProjectName          types.String `tfsdk:"project_name"`
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
				Description: "The scope to sync the secrets to. Must be 'project' for GitLab project variables.",
			},
			"project_id": schema.StringAttribute{
				Required:    true,
				Description: "The GitLab project ID to sync secrets to. This is the numeric project ID, not the project path.",
			},
			"project_name": schema.StringAttribute{
				Optional:    true,
				Description: "The GitLab project name for reference (optional, not used in API calls).",
			},
			"target_environment": schema.StringAttribute{
				Required:    true,
				Description: "The target environment for the GitLab CI/CD variables. Use '*' for all environments or specify a specific environment name.",
			},
			"should_protect_secrets": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to protect the GitLab CI/CD variables (only available in protected branches and tags).",
				Default:     booldefault.StaticBool(false),
			},
			"should_mask_secrets": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to mask the GitLab CI/CD variables in job logs.",
				Default:     booldefault.StaticBool(true),
			},
			"should_hide_secrets": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to hide the GitLab CI/CD variables from the UI (only shows variable names, not values).",
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

		ReadSyncOptionsForCreateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			syncOptionsMap := make(map[string]interface{})

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

		ReadSyncOptionsFromApi: func(ctx context.Context, secretSync infisicalclient.SecretSync) (types.Object, diag.Diagnostics) {
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

		ReadDestinationConfigForCreateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			destinationConfig := make(map[string]interface{})

			var gitlabCfg SecretSyncGitlabDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &gitlabCfg, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			// Validate scope
			if gitlabCfg.Scope.ValueString() != "project" {
				diags.AddError(
					"Invalid destination config",
					"Expected 'scope' to be 'project' for GitLab sync",
				)
				return nil, diags
			}

			destinationConfig["scope"] = gitlabCfg.Scope.ValueString()
			destinationConfig["projectId"] = gitlabCfg.ProjectId.ValueString()
			
			if !gitlabCfg.ProjectName.IsNull() && gitlabCfg.ProjectName.ValueString() != "" {
				destinationConfig["projectName"] = gitlabCfg.ProjectName.ValueString()
			}
			
			destinationConfig["targetEnvironment"] = gitlabCfg.TargetEnvironment.ValueString()
			
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

		ReadDestinationConfigForUpdateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel, _ SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			destinationConfig := make(map[string]interface{})

			var gitlabCfg SecretSyncGitlabDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &gitlabCfg, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			// Validate scope
			if gitlabCfg.Scope.ValueString() != "project" {
				diags.AddError(
					"Invalid destination config",
					"Expected 'scope' to be 'project' for GitLab sync",
				)
				return nil, diags
			}

			destinationConfig["scope"] = gitlabCfg.Scope.ValueString()
			destinationConfig["projectId"] = gitlabCfg.ProjectId.ValueString()
			
			if !gitlabCfg.ProjectName.IsNull() && gitlabCfg.ProjectName.ValueString() != "" {
				destinationConfig["projectName"] = gitlabCfg.ProjectName.ValueString()
			}
			
			destinationConfig["targetEnvironment"] = gitlabCfg.TargetEnvironment.ValueString()
			
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

		ReadDestinationConfigFromApi: func(ctx context.Context, secretSync infisicalclient.SecretSync) (types.Object, diag.Diagnostics) {
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
				"scope":                 types.StringType,
				"project_id":            types.StringType,
				"project_name":          types.StringType,
				"target_environment":    types.StringType,
				"should_protect_secrets": types.BoolType,
				"should_mask_secrets":    types.BoolType,
				"should_hide_secrets":    types.BoolType,
			}, destinationConfig)
		},
	}
}