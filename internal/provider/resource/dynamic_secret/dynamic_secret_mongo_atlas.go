package resource

import (
	"context"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type DynamicSecretMongoAtlasConfigurationModel struct {
	AdminPublicKey  types.String `tfsdk:"admin_public_key"`
	AdminPrivateKey types.String `tfsdk:"admin_private_key"`
	GroupId         types.String `tfsdk:"group_id"`
	Roles           types.List   `tfsdk:"roles"`
	Scopes          types.List   `tfsdk:"scopes"`
}

func NewDynamicSecretMongoAtlasResource() resource.Resource {
	return &DynamicSecretBaseResource{
		Provider:          infisical.DynamicSecretProviderMongoDBAtlas,
		ResourceTypeName:  "_dynamic_secret_mongo_atlas",
		DynamicSecretName: "MongoDB Atlas",
		ConfigurationAttributes: map[string]schema.Attribute{
			"admin_public_key": schema.StringAttribute{
				Required:    true,
				Description: "Admin user public api key",
			},
			"admin_private_key": schema.StringAttribute{
				Required:    true,
				Description: "Admin user private api key",
				Sensitive:   true,
			},
			"group_id": schema.StringAttribute{
				Required:    true,
				Description: "TUnique 24-hexadecimal digit string that identifies your project. This is same as project id",
			},
			"roles": schema.ListNestedAttribute{
				Required: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"database_name": schema.StringAttribute{
							Description: "Database to which the user is granted access privileges.",
							Required:    true,
						},
						"role_name": schema.StringAttribute{
							Description: "Human-readable label that identifies a group of privileges assigned to a database user. This value can either be a built-in role or a custom role. Refer to https://infisical.com/docs/api-reference/endpoints/dynamic-secrets/create#option-8 for supported options.",
							Required:    true,
						},
						"collection_name": schema.StringAttribute{
							Description: "Collection on which this role applies.",
							Optional:    true,
						},
					},
				},
			},
			"scopes": schema.ListNestedAttribute{
				Required: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Human-readable label that identifies the cluster or MongoDB Atlas Data Lake that this database user can access.",
							Required:    true,
						},
						"type": schema.StringAttribute{
							Description: "Category of resource that this database user can access. Supported options: CLUSTER, DATA_LAKE, STREAM",
							Required:    true,
						},
					},
				},
			},
		},

		ReadConfigurationFromPlan: func(ctx context.Context, plan DynamicSecretBaseResourceModel) (map[string]any, diag.Diagnostics) {
			configurationMap := make(map[string]any)
			var configuration DynamicSecretMongoAtlasConfigurationModel

			diags := plan.Configuration.As(ctx, &configuration, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			configurationMap["adminPublicKey"] = configuration.AdminPublicKey.ValueString()
			configurationMap["adminPrivateKey"] = configuration.AdminPrivateKey.ValueString()
			configurationMap["groupId"] = configuration.GroupId.ValueString()

			// Process roles list
			if !configuration.Roles.IsNull() && !configuration.Roles.IsUnknown() {
				var roles []any
				elements := configuration.Roles.Elements()
				for _, elem := range elements {
					if objVal, ok := elem.(types.Object); ok {
						attrs := objVal.Attributes()
						roleMap := map[string]any{
							"databaseName": attrs["database_name"].(types.String).ValueString(),
							"roleName":     attrs["role_name"].(types.String).ValueString(),
						}
						if collectionName, ok := attrs["collection_name"].(types.String); ok && !collectionName.IsNull() {
							roleMap["collectionName"] = collectionName.ValueString()
						}
						roles = append(roles, roleMap)
					}
				}
				configurationMap["roles"] = roles
			}

			// Process scopes list
			if !configuration.Scopes.IsNull() && !configuration.Scopes.IsUnknown() {
				var scopes []any
				elements := configuration.Scopes.Elements()
				for _, elem := range elements {
					if objVal, ok := elem.(types.Object); ok {
						attrs := objVal.Attributes()
						scopeMap := map[string]any{
							"name": attrs["name"].(types.String).ValueString(),
							"type": attrs["type"].(types.String).ValueString(),
						}
						scopes = append(scopes, scopeMap)
					}
				}
				configurationMap["scopes"] = scopes
			}

			return configurationMap, diags
		},

		ReadConfigurationFromApi: func(ctx context.Context, dynamicSecret infisical.DynamicSecret, configState types.Object) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			inputs := dynamicSecret.Inputs

			// Process roles from API
			var rolesList []attr.Value
			if rolesData, ok := inputs["roles"].([]any); ok {
				for _, role := range rolesData {
					if roleMap, ok := role.(map[string]any); ok {
						roleAttrs := map[string]attr.Value{
							"database_name": types.StringValue(getStringFromMap(roleMap, "databaseName")),
							"role_name":     types.StringValue(getStringFromMap(roleMap, "roleName")),
						}
						if collectionName := getStringFromMap(roleMap, "collectionName"); collectionName != "" {
							roleAttrs["collection_name"] = types.StringValue(collectionName)
						} else {
							roleAttrs["collection_name"] = types.StringNull()
						}
						roleObj, _ := types.ObjectValue(map[string]attr.Type{
							"database_name":   types.StringType,
							"role_name":       types.StringType,
							"collection_name": types.StringType,
						}, roleAttrs)
						rolesList = append(rolesList, roleObj)
					}
				}
			}
			rolesListValue, _ := types.ListValue(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"database_name":   types.StringType,
					"role_name":       types.StringType,
					"collection_name": types.StringType,
				},
			}, rolesList)

			// Process scopes from API
			var scopesList []attr.Value
			if scopesData, ok := inputs["scopes"].([]any); ok {
				for _, scope := range scopesData {
					if scopeMap, ok := scope.(map[string]any); ok {
						scopeAttrs := map[string]attr.Value{
							"name": types.StringValue(getStringFromMap(scopeMap, "name")),
							"type": types.StringValue(getStringFromMap(scopeMap, "type")),
						}
						scopeObj, _ := types.ObjectValue(map[string]attr.Type{
							"name": types.StringType,
							"type": types.StringType,
						}, scopeAttrs)
						scopesList = append(scopesList, scopeObj)
					}
				}
			}
			scopesListValue, _ := types.ListValue(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"name": types.StringType,
					"type": types.StringType,
				},
			}, scopesList)

			configAttrs := map[string]attr.Value{
				"admin_public_key":  types.StringValue(getStringFromMap(inputs, "adminPublicKey")),
				"admin_private_key": types.StringValue(getStringFromMap(inputs, "adminPrivateKey")),
				"group_id":          types.StringValue(getStringFromMap(inputs, "groupId")),
				"roles":             rolesListValue,
			}

			if len(scopesList) > 0 {
				configAttrs["scopes"] = scopesListValue
			} else {
				configAttrs["scopes"] = types.ListNull(types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"name": types.StringType,
						"type": types.StringType,
					},
				})
			}

			configType := map[string]attr.Type{
				"admin_public_key":  types.StringType,
				"admin_private_key": types.StringType,
				"group_id":          types.StringType,
				"roles": types.ListType{
					ElemType: types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"database_name":   types.StringType,
							"role_name":       types.StringType,
							"collection_name": types.StringType,
						},
					},
				},
				"scopes": types.ListType{
					ElemType: types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"name": types.StringType,
							"type": types.StringType,
						},
					},
				},
			}

			configObject, diags := types.ObjectValue(configType, configAttrs)
			if diags.HasError() {
				return types.ObjectNull(configState.AttributeTypes(ctx)), diags
			}

			return configObject, diags
		},
	}
}

func getStringFromMap(m map[string]any, key string) string {
	if val, ok := m[key]; ok {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return ""
}
