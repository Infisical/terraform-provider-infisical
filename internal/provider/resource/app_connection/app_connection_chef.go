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

type AppConnectionChefCredentialsModel struct {
	ServerUrl  types.String `tfsdk:"server_url"`
	OrgName    types.String `tfsdk:"org_name"`
	UserName   types.String `tfsdk:"user_name"`
	PrivateKey types.String `tfsdk:"private_key"`
}

const ChefAppConnectionUserKeyMethod = "user-key"

func NewAppConnectionChefResource() resource.Resource {
	return &AppConnectionBaseResource{
		App:               infisical.AppConnectionAppChef,
		AppConnectionName: "Chef",
		ResourceTypeName:  "_app_connection_chef",
		SupportsGateway:   true,
		AllowedMethods:    []string{ChefAppConnectionUserKeyMethod},
		CredentialsAttributes: map[string]schema.Attribute{
			"server_url": schema.StringAttribute{
				Optional:    true,
				Description: "The URL of the Chef Server to connect to, e.g. `https://chef.example.com`, defaulting to the hosted Chef API when omitted.",
			},
			"org_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Chef organization to connect to.",
			},
			"user_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Chef user to authenticate as.",
			},
			"private_key": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "The PEM-encoded private key of the Chef user.",
			},
		},
		ReadCredentialsForCreateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			credentialsConfig := make(map[string]any)

			var credentials AppConnectionChefCredentialsModel
			diags := plan.Credentials.As(ctx, &credentials, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != ChefAppConnectionUserKeyMethod {
				diags.AddError(
					"Unable to create Chef app connection",
					"Invalid method. Only user-key method is supported",
				)
				return nil, diags
			}

			if credentials.OrgName.IsNull() || credentials.OrgName.ValueString() == "" {
				diags.AddError(
					"Unable to create Chef app connection",
					"Org name field must be defined in user-key method",
				)
				return nil, diags
			}

			if credentials.UserName.IsNull() || credentials.UserName.ValueString() == "" {
				diags.AddError(
					"Unable to create Chef app connection",
					"User name field must be defined in user-key method",
				)
				return nil, diags
			}

			if credentials.PrivateKey.IsNull() || credentials.PrivateKey.ValueString() == "" {
				diags.AddError(
					"Unable to create Chef app connection",
					"Private key field must be defined in user-key method",
				)
				return nil, diags
			}

			if !credentials.ServerUrl.IsNull() && credentials.ServerUrl.ValueString() != "" {
				credentialsConfig["serverUrl"] = credentials.ServerUrl.ValueString()
			}
			credentialsConfig["orgName"] = credentials.OrgName.ValueString()
			credentialsConfig["userName"] = credentials.UserName.ValueString()
			credentialsConfig["privateKey"] = credentials.PrivateKey.ValueString()

			return credentialsConfig, diags
		},
		ReadCredentialsForUpdateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel, state AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			credentialsConfig := make(map[string]any)

			var credentialsFromPlan AppConnectionChefCredentialsModel
			diags := plan.Credentials.As(ctx, &credentialsFromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var credentialsFromState AppConnectionChefCredentialsModel
			diags = state.Credentials.As(ctx, &credentialsFromState, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != ChefAppConnectionUserKeyMethod {
				diags.AddError(
					"Unable to update Chef app connection",
					"Invalid method. Only user-key method is supported",
				)
				return nil, diags
			}

			serverUrl := credentialsFromPlan.ServerUrl
			if credentialsFromPlan.ServerUrl.IsUnknown() {
				serverUrl = credentialsFromState.ServerUrl
			}

			orgName := credentialsFromPlan.OrgName
			if credentialsFromPlan.OrgName.IsUnknown() {
				orgName = credentialsFromState.OrgName
			}

			userName := credentialsFromPlan.UserName
			if credentialsFromPlan.UserName.IsUnknown() {
				userName = credentialsFromState.UserName
			}

			privateKey := credentialsFromPlan.PrivateKey
			if credentialsFromPlan.PrivateKey.IsUnknown() {
				privateKey = credentialsFromState.PrivateKey
			}

			if orgName.IsNull() || orgName.ValueString() == "" {
				diags.AddError(
					"Unable to update Chef app connection",
					"Org name field must be defined in user-key method",
				)
				return nil, diags
			}

			if userName.IsNull() || userName.ValueString() == "" {
				diags.AddError(
					"Unable to update Chef app connection",
					"User name field must be defined in user-key method",
				)
				return nil, diags
			}

			if privateKey.IsNull() || privateKey.ValueString() == "" {
				diags.AddError(
					"Unable to update Chef app connection",
					"Private key field must be defined in user-key method",
				)
				return nil, diags
			}

			if !serverUrl.IsNull() && serverUrl.ValueString() != "" {
				credentialsConfig["serverUrl"] = serverUrl.ValueString()
			}
			credentialsConfig["orgName"] = orgName.ValueString()
			credentialsConfig["userName"] = userName.ValueString()
			credentialsConfig["privateKey"] = privateKey.ValueString()

			return credentialsConfig, diags
		},
		OverwriteCredentialsFields: func(state *AppConnectionBaseResourceModel) diag.Diagnostics {
			credentialsConfig := map[string]attr.Value{
				"server_url":  types.StringNull(),
				"org_name":    types.StringNull(),
				"user_name":   types.StringNull(),
				"private_key": types.StringNull(),
			}

			var diags diag.Diagnostics
			state.Credentials, diags = types.ObjectValue(map[string]attr.Type{
				"server_url":  types.StringType,
				"org_name":    types.StringType,
				"user_name":   types.StringType,
				"private_key": types.StringType,
			}, credentialsConfig)

			return diags
		},
	}
}
