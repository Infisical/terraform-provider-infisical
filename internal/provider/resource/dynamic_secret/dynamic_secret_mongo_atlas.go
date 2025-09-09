package resource

import (
	"context"
	"strconv"
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
		Provider:          infisical.DynamicSecretProviderMongoAtlas,
		ResourceTypeName:  "_dynamic_secret_mongo_atlas",
		DynamicSecretName: "MongoDB Atlas",
		ConfigurationAttributes: map[string]schema.Attribute{
			"admin_public_key": schema.StringAttribute{
				Required:    true,
				Description: "Admin user public API key",
			},
			"admin_private_key": schema.StringAttribute{
				Required:    true,
				Description: "Admin user private API key",
				Sensitive:   true,
			},
			"group_id": schema.StringAttribute{
				Required:    true,
				Description: "Unique 24-hexadecimal digit string that identifies your project. This is the same as the project ID.",
			},
			"roles": schema.ListNestedAttribute{
				Required: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"collection_name": schema.StringAttribute{
							Optional:    true,
							Description: "Collection on which this role applies.",
						},
						"database_name": schema.StringAttribute{
							Required:    true,
							Description: "Database to which the user is granted access privileges.",
						},
						"role_name": schema.StringAttribute{
							Required:    true,
							Description: "Human-readable label that identifies a group of privileges assigned to a database user. This value can either be a built-in role or a custom role.",
						},
					},
				},
			},
			"scopes": schema.ListNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "Human-readable label that identifies the cluster or MongoDB Atlas Data Lake that this database user can access.",
						},
						"type": schema.StringAttribute{
							Required:    true,
							Description: "Category of resource that this database user can access. Supported options: CLUSTER, DATA_LAKE, STREAM",
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

			if !configuration.Roles.IsNull() && !configuration.Roles.IsUnknown() {
				var roles []any
				elements := configuration.Roles.Elements()
				for _, elem := range elements {
					if objVal, ok := elem.(types.Object); ok {
						attrs := objVal.Attributes()
						roleMap := map[string]any{}
						if collectionName, ok := attrs["collection_name"].(types.String); ok && !collectionName.IsNull() {
							roleMap["collectionName"] = collectionName.ValueString()
						}
						if dbName, ok := attrs["database_name"].(types.String); ok {
							roleMap["databaseName"] = dbName.ValueString()
						}
						if roleName, ok := attrs["role_name"].(types.String); ok {
							roleMap["roleName"] = roleName.ValueString()
						}
						roles = append(roles, roleMap)
					}
				}
				configurationMap["roles"] = roles
			}

			scopes := []any{}
			if !configuration.Scopes.IsNull() && !configuration.Scopes.IsUnknown() {
				elements := configuration.Scopes.Elements()
				for _, elem := range elements {
					if objVal, ok := elem.(types.Object); ok {
						attrs := objVal.Attributes()
						scopeMap := map[string]any{}
						if name, ok := attrs["name"].(types.String); ok {
							scopeMap["name"] = name.ValueString()
						}
						if typeVal, ok := attrs["type"].(types.String); ok {
							scopeMap["type"] = typeVal.ValueString()
						}
						scopes = append(scopes, scopeMap)
					}
				}
			}
			configurationMap["scopes"] = scopes

			return configurationMap, diags
		},

		ReadConfigurationFromApi: func(ctx context.Context, dynamicSecret infisical.DynamicSecret, configState types.Object) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			var currentState DynamicSecretMongoAtlasConfigurationModel
			stateDiags := configState.As(ctx, &currentState, basetypes.ObjectAsOptions{})
			diags.Append(stateDiags...)
			if diags.HasError() {
				return types.ObjectNull(configState.AttributeTypes(ctx)), diags
			}

			adminPublicKey, ok := dynamicSecret.Inputs["adminPublicKey"].(string)
			if !ok {
				diags.AddError(
					"Invalid adminPublicKey type",
					"Expected 'adminPublicKey' to be a string but got something else.",
				)
			}
			adminPrivateKey, ok := dynamicSecret.Inputs["adminPrivateKey"].(string)
			if !ok {
				diags.AddError(
					"Invalid adminPrivateKey type",
					"Expected 'adminPrivateKey' to be a string but got something else.",
				)
			}
			groupId, ok := dynamicSecret.Inputs["groupId"].(string)
			if !ok {
				diags.AddError(
					"Invalid groupId type",
					"Expected 'groupId' to be a string but got something else.",
				)
			}

			if diags.HasError() {
				return types.ObjectNull(configState.AttributeTypes(ctx)), diags
			}

			configuration := map[string]attr.Value{
				"admin_public_key":  types.StringValue(adminPublicKey),
				"admin_private_key": types.StringValue(adminPrivateKey),
				"group_id":          types.StringValue(groupId),
			}

			rolesRaw, ok := dynamicSecret.Inputs["roles"].([]any)
			if !ok {
				diags.AddError(
					"Invalid roles type",
					"Expected 'roles' to be a list but got something else.",
				)
				return types.ObjectNull(configState.AttributeTypes(ctx)), diags
			}

			var rolesList []attr.Value
			for i, roleRaw := range rolesRaw {
				roleMap, ok := roleRaw.(map[string]any)
				if !ok {
					diags.AddError(
						"Invalid role element type",
						"Expected role at index "+strconv.Itoa(i)+" to be an object but got something else.",
					)
					continue
				}

				databaseName, ok := roleMap["databaseName"].(string)
				if !ok {
					diags.AddError(
						"Invalid databaseName type in role",
						"Expected 'databaseName' to be a string but got something else.",
					)
				}
				roleName, ok := roleMap["roleName"].(string)
				if !ok {
					diags.AddError(
						"Invalid roleName type in role",
						"Expected 'roleName' to be a string but got something else.",
					)
				}
				if diags.HasError() {
					continue
				}

				roleAttrs := map[string]attr.Value{
					"database_name": types.StringValue(databaseName),
					"role_name":     types.StringValue(roleName),
				}

				if collectionName, ok := roleMap["collectionName"].(string); ok && collectionName != "" {
					roleAttrs["collection_name"] = types.StringValue(collectionName)
				} else {
					roleAttrs["collection_name"] = types.StringNull()
				}

				roleObj, roleObjDiags := types.ObjectValue(map[string]attr.Type{
					"database_name":   types.StringType,
					"role_name":       types.StringType,
					"collection_name": types.StringType,
				}, roleAttrs)
				diags.Append(roleObjDiags...)
				rolesList = append(rolesList, roleObj)
			}

			if diags.HasError() {
				return types.ObjectNull(configState.AttributeTypes(ctx)), diags
			}

			rolesListValue, rolesListDiags := types.ListValue(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"database_name":   types.StringType,
					"role_name":       types.StringType,
					"collection_name": types.StringType,
				},
			}, rolesList)
			diags.Append(rolesListDiags...)
			configuration["roles"] = rolesListValue

			var scopesRaw []any
			if scopes, ok := dynamicSecret.Inputs["scopes"].([]any); ok {
				scopesRaw = scopes
			}

			var scopesList []attr.Value
			for i, scopeRaw := range scopesRaw {
				scopeMap, ok := scopeRaw.(map[string]any)
				if !ok {
					diags.AddError(
						"Invalid scope element type",
						"Expected scope at index "+strconv.Itoa(i)+" to be an object but got something else.",
					)
					continue
				}

				name, ok := scopeMap["name"].(string)
				if !ok {
					diags.AddError(
						"Invalid name type in scope",
						"Expected 'name' to be a string but got something else.",
					)
				}
				scopeType, ok := scopeMap["type"].(string)
				if !ok {
					diags.AddError(
						"Invalid type type in scope",
						"Expected 'type' to be a string but got something else.",
					)
				}
				if diags.HasError() {
					continue
				}

				scopeAttrs := map[string]attr.Value{
					"name": types.StringValue(name),
					"type": types.StringValue(scopeType),
				}

				scopeObj, scopeObjDiags := types.ObjectValue(map[string]attr.Type{
					"name": types.StringType,
					"type": types.StringType,
				}, scopeAttrs)
				diags.Append(scopeObjDiags...)
				scopesList = append(scopesList, scopeObj)
			}

			if diags.HasError() {
				return types.ObjectNull(configState.AttributeTypes(ctx)), diags
			}

			scopesListValue, scopesListDiags := types.ListValue(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"name": types.StringType,
					"type": types.StringType,
				},
			}, scopesList)
			diags.Append(scopesListDiags...)
			configuration["scopes"] = scopesListValue

			if diags.HasError() {
				return types.ObjectNull(configState.AttributeTypes(ctx)), diags
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

			configObject, objDiags := types.ObjectValue(configType, configuration)
			diags.Append(objDiags...)
			if diags.HasError() {
				return types.ObjectNull(configState.AttributeTypes(ctx)), diags
			}

			return configObject, diags
		},
	}
}
