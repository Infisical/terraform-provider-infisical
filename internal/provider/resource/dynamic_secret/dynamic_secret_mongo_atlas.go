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
	PublicKey  types.String `tfsdk:"public_key"`
	PrivateKey types.String `tfsdk:"private_key"`
	GroupId    types.String `tfsdk:"group_id"`
	Roles      types.String `tfsdk:"roles"`
	Scopes     types.String `tfsdk:"scopes"`
}

func NewDynamicSecretMongoAtlasResource() resource.Resource {
	return &DynamicSecretBaseResource{
		Provider:          infisical.DynamicSecretProviderMongoDBAtlas,
		ResourceTypeName:  "_dynamic_secret_mongo_atlas",
		DynamicSecretName: "MongoDB Atlas",
		ConfigurationAttributes: map[string]schema.Attribute{
			"public_key": schema.StringAttribute{
				Required:    true,
				Description: "The MongoDB Atlas public API key for authentication.",
			},
			"private_key": schema.StringAttribute{
				Required:    true,
				Description: "The MongoDB Atlas private API key for authentication.",
				Sensitive:   true,
			},
			"group_id": schema.StringAttribute{
				Required:    true,
				Description: "The MongoDB Atlas project ID (also known as group ID).",
			},
			"roles": schema.StringAttribute{
				Required:    true,
				Description: "Comma-separated list of roles to assign to the created database user. Example: 'readWrite@mydb,read@admin'",
			},
			"scopes": schema.StringAttribute{
				Optional:    true,
				Description: "Comma-separated list of cluster names or data lake names to restrict the database user to. If not specified, the user will have access to all clusters and data lakes in the project.",
			},
		},

		ReadConfigurationFromPlan: func(ctx context.Context, plan DynamicSecretBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			configurationMap := make(map[string]interface{})
			var configuration DynamicSecretMongoAtlasConfigurationModel

			diags := plan.Configuration.As(ctx, &configuration, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			configurationMap["publicKey"] = configuration.PublicKey.ValueString()
			configurationMap["privateKey"] = configuration.PrivateKey.ValueString()
			configurationMap["groupId"] = configuration.GroupId.ValueString()
			configurationMap["roles"] = configuration.Roles.ValueString()

			if !configuration.Scopes.IsNull() && !configuration.Scopes.IsUnknown() {
				configurationMap["scopes"] = configuration.Scopes.ValueString()
			}

			return configurationMap, diags
		},

		ReadConfigurationFromApi: func(ctx context.Context, dynamicSecret infisical.DynamicSecret, configState types.Object) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			inputs := dynamicSecret.Inputs

			configAttrs := map[string]attr.Value{
				"public_key":  types.StringValue(getStringFromMap(inputs, "publicKey")),
				"private_key": types.StringValue(getStringFromMap(inputs, "privateKey")),
				"group_id":    types.StringValue(getStringFromMap(inputs, "groupId")),
				"roles":       types.StringValue(getStringFromMap(inputs, "roles")),
			}

			if scopes := getStringFromMap(inputs, "scopes"); scopes != "" {
				configAttrs["scopes"] = types.StringValue(scopes)
			} else {
				configAttrs["scopes"] = types.StringNull()
			}

			configType := map[string]attr.Type{
				"public_key":  types.StringType,
				"private_key": types.StringType,
				"group_id":    types.StringType,
				"roles":       types.StringType,
				"scopes":      types.StringType,
			}

			configObject, diags := types.ObjectValue(configType, configAttrs)
			if diags.HasError() {
				return types.ObjectNull(configState.AttributeTypes(ctx)), diags
			}

			return configObject, diags
		},
	}
}

func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return ""
}
