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

type GithubSyncScope string
type GithubSyncVisibility string

const (
	// We also support `private` and `all`, but they require no extra logic code-wise.
	GithubSyncVisibilitySelected GithubSyncVisibility = "selected"
)

const (
	GithubSyncScopeRepository            GithubSyncScope = "repository"
	GithubSyncScopeRepositoryEnvironment GithubSyncScope = "repository-environment"
	GithubSyncScopeOrganization          GithubSyncScope = "organization"
)

func verifyDestinationConfigState(destinationConfig map[string]attr.Value, diags *diag.Diagnostics) bool {
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

	scope := GithubSyncScope(scopeVal.ValueString())
	var requiredFields []string

	switch scope {
	case GithubSyncScopeRepository:
		requiredFields = []string{"scope", "repository_owner", "repository_name"}

	case GithubSyncScopeRepositoryEnvironment:
		requiredFields = []string{"scope", "repository_environment", "repository_owner", "repository_name"}

	case GithubSyncScopeOrganization:
		requiredFields = []string{"scope", "repository_owner", "visibility"}

		// For organization scope with "selected" visibility, selected_repository_ids is required
		if visibilityAttr, exists := destinationConfig["visibility"]; exists {
			if visibilityVal, ok := visibilityAttr.(types.String); ok && !visibilityVal.IsNull() && !visibilityVal.IsUnknown() {
				if GithubSyncVisibility(visibilityVal.ValueString()) == GithubSyncVisibilitySelected {
					requiredFields = append(requiredFields, "selected_repository_ids")
				}
			}
		}

	default:
		diags.AddError("Invalid destination config", fmt.Sprintf("Invalid scope '%s'", scope))
		return false
	}

	// Check required fields are not empty
	for _, field := range requiredFields {
		value, exists := destinationConfig[field]
		if !exists {
			diags.AddError("Invalid destination config", fmt.Sprintf("Expected '%s' to be present when scope is %s", field, scope))
			return false
		}

		// Check if the value is null, unknown, or empty based on its type
		if terraform.IsAttrValueEmpty(value) {
			diags.AddError("Invalid destination config", fmt.Sprintf("Expected '%s' to be set when scope is %s", field, scope))
			return false
		}
	}

	// Check for unexpected fields
	allowedFieldsMap := make(map[string]bool)
	for _, field := range requiredFields {
		allowedFieldsMap[field] = true
	}

	for field := range destinationConfig {
		if !allowedFieldsMap[field] {
			if terraform.IsAttrValueEmpty(destinationConfig[field]) {
				continue
			}

			diags.AddError("Invalid destination config", fmt.Sprintf("Unexpected field '%s' for scope '%s'. Supported destination_config fields are: %v", field, scope, requiredFields))
			return false
		}
	}

	return true
}

// SecretSyncGithubDestinationConfigModel describes the data source data model.

type SecretSyncGithubDestinationConfigModel struct {
	Scope types.String `tfsdk:"scope"` // repository|organization|repository-environment

	RepositoryOwner types.String `tfsdk:"repository_owner"`
	RepositoryName  types.String `tfsdk:"repository_name"`

	Visibility            types.String `tfsdk:"visibility"`
	SelectedRepositoryIds types.List   `tfsdk:"selected_repository_ids"`
	RepositoryEnvironment types.String `tfsdk:"repository_environment"`
}

type SecretSyncGithubSyncOptionsModel struct {
	InitialSyncBehavior   types.String `tfsdk:"initial_sync_behavior"`
	DisableSecretDeletion types.Bool   `tfsdk:"disable_secret_deletion"`
	KeySchema             types.String `tfsdk:"key_schema"`
}

func NewSecretSyncGithubResource() resource.Resource {
	return &SecretSyncBaseResource{
		App:              infisical.SecretSyncAppGithub,
		SyncName:         "Github",
		ResourceTypeName: "_secret_sync_github",
		AppConnection:    infisical.AppConnectionAppGithub,
		DestinationConfigAttributes: map[string]schema.Attribute{
			"scope": schema.StringAttribute{
				Required:    true,
				Description: "The scope to sync the secrets to, repository|organization",
			},
			"repository_owner": schema.StringAttribute{
				Optional:    true,
				Description: "The owner of the Github repository, required if scope is `repository`, `repository-environment`, or `organization`. This is the organization name, or the username for personal repositories. As an example if you have a repository called Infisical/go-sdk, you would only need to provide `Infisical` here.",
			},
			"repository_name": schema.StringAttribute{
				Optional:    true,
				Description: "The repository to sync the secrets to, required if scope is `repository` or `repository-environment`. This is only the name of the repository, without the repository owner included. As an example if you have a repository called Infisical/go-sdk, you would only need to provide `go-sdk` here.",
			},
			"visibility": schema.StringAttribute{
				Optional:    true,
				Description: "The visibility of the Github repository, required if scope is `organization`. Accepted values are: `all`|`private`|`selected`",
			},
			"selected_repository_ids": schema.ListAttribute{
				Optional:    true,
				ElementType: types.Int64Type,
				Description: "The repository ids to sync the secrets to, required if scope is `organization` and the visibility field is set to `selected`",
			},
			"repository_environment": schema.StringAttribute{
				Optional:    true,
				Description: "The environment to sync the secrets to, required if scope is `repository-environment`",
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
				Description: "When set to true, Infisical will not remove secrets from Github. Enable this option if you intend to manage some secrets manually outside of Infisical.",
				Default:     booldefault.StaticBool(false),
			},
			"key_schema": schema.StringAttribute{
				Optional:    true,
				Description: "The format to use for structuring secret keys in the Github destination.",
			},
		},

		ReadSyncOptionsForCreateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			syncOptionsMap := make(map[string]interface{})

			var syncOptions SecretSyncGithubSyncOptionsModel
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

			var syncOptions SecretSyncGithubSyncOptionsModel
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

			var githubCfg SecretSyncGithubDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &githubCfg, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			rawGithubCfg := map[string]attr.Value{
				"scope":                   githubCfg.Scope,
				"repository_owner":        githubCfg.RepositoryOwner,
				"repository_name":         githubCfg.RepositoryName,
				"repository_environment":  githubCfg.RepositoryEnvironment,
				"visibility":              githubCfg.Visibility,
				"selected_repository_ids": githubCfg.SelectedRepositoryIds,
			}

			if !verifyDestinationConfigState(rawGithubCfg, &diags) {
				return nil, diags
			}

			destinationConfig["scope"] = githubCfg.Scope.ValueString()

			if GithubSyncScope(githubCfg.Scope.ValueString()) == GithubSyncScopeRepository {
				destinationConfig["owner"] = githubCfg.RepositoryOwner.ValueString()
				destinationConfig["repo"] = githubCfg.RepositoryName.ValueString()
			}

			if GithubSyncScope(githubCfg.Scope.ValueString()) == GithubSyncScopeRepositoryEnvironment {
				destinationConfig["env"] = githubCfg.RepositoryEnvironment.ValueString()
				destinationConfig["owner"] = githubCfg.RepositoryOwner.ValueString()
				destinationConfig["repo"] = githubCfg.RepositoryName.ValueString()
			}

			if GithubSyncScope(githubCfg.Scope.ValueString()) == GithubSyncScopeOrganization {
				if GithubSyncVisibility(githubCfg.Visibility.ValueString()) == GithubSyncVisibilitySelected {
					if len(githubCfg.SelectedRepositoryIds.Elements()) == 0 {
						diags.AddError("Invalid selected repository ids", "Expected 'selected_repository_ids' to be set with at least one repository when scope is organization and visibility is `selected`")
						return nil, diags
					}

					selectedRepositoryIds := make([]int64, 0)
					for _, id := range githubCfg.SelectedRepositoryIds.Elements() {
						intVal, ok := id.(types.Int64)
						if !ok {
							diags.AddError(
								"Invalid repository ID type",
								"Expected repository ID to be an integer",
							)
							return nil, diags
						}
						selectedRepositoryIds = append(selectedRepositoryIds, intVal.ValueInt64())
					}

					destinationConfig["selectedRepositoryIds"] = selectedRepositoryIds
				}

				destinationConfig["org"] = githubCfg.RepositoryOwner.ValueString()
				destinationConfig["visibility"] = githubCfg.Visibility.ValueString()
			}

			return destinationConfig, diags
		},
		ReadDestinationConfigForUpdateFromPlan: func(ctx context.Context, plan SecretSyncBaseResourceModel, _ SecretSyncBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			destinationConfig := make(map[string]interface{})

			var githubCfg SecretSyncGithubDestinationConfigModel
			diags := plan.DestinationConfig.As(ctx, &githubCfg, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			rawGithubCfg := map[string]attr.Value{
				"scope":                   githubCfg.Scope,
				"repository_owner":        githubCfg.RepositoryOwner,
				"repository_name":         githubCfg.RepositoryName,
				"repository_environment":  githubCfg.RepositoryEnvironment,
				"visibility":              githubCfg.Visibility,
				"selected_repository_ids": githubCfg.SelectedRepositoryIds,
			}

			if !verifyDestinationConfigState(rawGithubCfg, &diags) {
				return nil, diags
			}

			destinationConfig["scope"] = githubCfg.Scope.ValueString()

			if GithubSyncScope(githubCfg.Scope.ValueString()) == GithubSyncScopeRepository {

				destinationConfig["owner"] = githubCfg.RepositoryOwner.ValueString()
				destinationConfig["repo"] = githubCfg.RepositoryName.ValueString()
			}

			if GithubSyncScope(githubCfg.Scope.ValueString()) == GithubSyncScopeRepositoryEnvironment {
				destinationConfig["env"] = githubCfg.RepositoryEnvironment.ValueString()
				destinationConfig["owner"] = githubCfg.RepositoryOwner.ValueString()
				destinationConfig["repo"] = githubCfg.RepositoryName.ValueString()
			}

			if GithubSyncScope(githubCfg.Scope.ValueString()) == GithubSyncScopeOrganization {

				if GithubSyncVisibility(githubCfg.Visibility.ValueString()) == GithubSyncVisibilitySelected {
					if len(githubCfg.SelectedRepositoryIds.Elements()) == 0 {
						diags.AddError("Invalid selected repository ids", "Expected 'selected_repository_ids' to be set with at least one repository when scope is organization and visibility is `selected`")
						return nil, diags
					}

					selectedRepositoryIds := make([]int64, 0)
					for _, id := range githubCfg.SelectedRepositoryIds.Elements() {
						intVal, ok := id.(types.Int64)
						if !ok {
							diags.AddError(
								"Invalid repository ID type",
								"Expected repository ID to be an integer",
							)
							return nil, diags
						}
						selectedRepositoryIds = append(selectedRepositoryIds, intVal.ValueInt64())
					}

					destinationConfig["selectedRepositoryIds"] = selectedRepositoryIds
				}

				destinationConfig["org"] = githubCfg.RepositoryOwner.ValueString()
				destinationConfig["visibility"] = githubCfg.Visibility.ValueString()
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
				"scope":                   types.StringValue(scopeVal),
				"repository_owner":        types.StringNull(),
				"repository_name":         types.StringNull(),
				"repository_environment":  types.StringNull(),
				"visibility":              types.StringNull(),
				"selected_repository_ids": types.ListNull(types.Int64Type),
			}

			if GithubSyncScope(scopeVal) == GithubSyncScopeRepository {
				ownerVal, ok := secretSync.DestinationConfig["owner"].(string)
				if !ok {
					diags.AddError(
						"Invalid owner type",
						"Expected 'owner' to be a string but got something else",
					)
					return types.ObjectNull(map[string]attr.Type{}), diags
				}

				repoVal, ok := secretSync.DestinationConfig["repo"].(string)
				if !ok {
					diags.AddError(
						"Invalid repo type",
						"Expected 'repo' to be a string but got something else",
					)
					return types.ObjectNull(map[string]attr.Type{}), diags
				}

				destinationConfig["repository_owner"] = types.StringValue(ownerVal)
				destinationConfig["repository_name"] = types.StringValue(repoVal)
			}

			if GithubSyncScope(scopeVal) == GithubSyncScopeRepositoryEnvironment {
				envVal, ok := secretSync.DestinationConfig["env"].(string)
				if !ok {
					diags.AddError(
						"Invalid env type",
						"Expected 'env' to be a string but got something else",
					)
					return types.ObjectNull(map[string]attr.Type{}), diags
				}

				ownerVal, ok := secretSync.DestinationConfig["owner"].(string)
				if !ok {
					diags.AddError(
						"Invalid owner type",
						"Expected 'owner' to be a string but got something else",
					)
					return types.ObjectNull(map[string]attr.Type{}), diags
				}

				repoVal, ok := secretSync.DestinationConfig["repo"].(string)
				if !ok {
					diags.AddError(
						"Invalid repo type",
						"Expected 'repo' to be a string but got something else",
					)
					return types.ObjectNull(map[string]attr.Type{}), diags
				}

				destinationConfig["repository_environment"] = types.StringValue(envVal)
				destinationConfig["repository_owner"] = types.StringValue(ownerVal)
				destinationConfig["repository_name"] = types.StringValue(repoVal)
			}

			if GithubSyncScope(scopeVal) == GithubSyncScopeOrganization {
				orgVal, ok := secretSync.DestinationConfig["org"].(string)
				if !ok {
					diags.AddError(
						"Invalid org type",
						"Expected 'org' to be a string but got something else",
					)
				}

				visibilityVal, ok := secretSync.DestinationConfig["visibility"].(string)
				if !ok {
					diags.AddError(
						"Invalid visibility type",
						"Expected 'visibility' to be a string but got something else",
					)
					return types.ObjectNull(map[string]attr.Type{}), diags
				}

				if GithubSyncVisibility(visibilityVal) == GithubSyncVisibilitySelected {

					selectedRepositoryIdsRaw, exists := secretSync.DestinationConfig["selectedRepositoryIds"]
					if !exists {
						diags.AddError(
							"Missing selected repository ids",
							"Expected 'selectedRepositoryIds' to be present",
						)
						return types.ObjectNull(map[string]attr.Type{}), diags
					}

					selectedRepositoryIdsSlice, ok := selectedRepositoryIdsRaw.([]interface{})
					if !ok {
						diags.AddError(
							"Invalid selected repository ids type",
							"Expected 'selectedRepositoryIds' to be an array but got something else: "+fmt.Sprintf("%T", selectedRepositoryIdsRaw),
						)
						return types.ObjectNull(map[string]attr.Type{}), diags
					}

					selectedRepositoryIds := make([]int64, len(selectedRepositoryIdsSlice))

					// Go translates numbers to floating point numbers when they're being taken from a raw interface{}, so we need to do do some magic to get it in the right number format
					for i, idRaw := range selectedRepositoryIdsSlice {
						switch id := idRaw.(type) {
						case int64:
							selectedRepositoryIds[i] = id
						case float64:
							selectedRepositoryIds[i] = int64(id)
						case int:
							selectedRepositoryIds[i] = int64(id)
						default:
							diags.AddError(
								"Invalid repository id type",
								fmt.Sprintf("Expected repository ID to be a number but got %T: %v", id, id),
							)
							return types.ObjectNull(map[string]attr.Type{}), diags
						}
					}

					if len(selectedRepositoryIds) == 0 {
						diags.AddError(
							"Invalid selected repository ids",
							"Expected 'selectedRepositoryIds' to be non-empty when scope is organization and visibility is `selected`",
						)
						return types.ObjectNull(map[string]attr.Type{}), diags
					}

					repoIdValues := make([]attr.Value, len(selectedRepositoryIds))
					for i, id := range selectedRepositoryIds {
						repoIdValues[i] = types.Int64Value(id)
					}

					listVal, listDiags := types.ListValue(types.Int64Type, repoIdValues)
					diags.Append(listDiags...)
					if diags.HasError() {
						return types.ObjectNull(map[string]attr.Type{}), diags
					}

					destinationConfig["selected_repository_ids"] = listVal
				}

				destinationConfig["repository_owner"] = types.StringValue(orgVal)
				destinationConfig["visibility"] = types.StringValue(visibilityVal)
			}

			if !verifyDestinationConfigState(destinationConfig, &diags) {
				return types.ObjectNull(map[string]attr.Type{}), diags
			}

			return types.ObjectValue(map[string]attr.Type{
				"scope":                  types.StringType,
				"repository_owner":       types.StringType,
				"repository_name":        types.StringType,
				"visibility":             types.StringType,
				"repository_environment": types.StringType,
				"selected_repository_ids": types.ListType{
					ElemType: types.Int64Type,
				},
			}, destinationConfig)
		},
	}
}
