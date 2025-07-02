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

// AppConnection1PasswordCredentialsModel describes the data source data model.
type AppConnection1PasswordCredentialsModel struct {
	InstanceUrl types.String `tfsdk:"instance_url"`
	ApiToken    types.String `tfsdk:"api_token"`
}

const AppConnection1PasswordAuthMethodApiToken = "api-token"

func NewAppConnection1PasswordResource() resource.Resource {
	return &AppConnectionBaseResource{
		App:               infisical.AppConnectionApp1Password,
		AppConnectionName: "1Password",
		ResourceTypeName:  "_app_connection_1password",
		AllowedMethods:    []string{AppConnection1PasswordAuthMethodApiToken},
		CredentialsAttributes: map[string]schema.Attribute{
			"instance_url": schema.StringAttribute{
				Optional:    true,
				Description: "The URL of the 1Password Connect instance to connect to. For more details, refer to the documentation here infisical.com/docs/integrations/app-connections/1password",
				Sensitive:   true,
			},
			"api_token": schema.StringAttribute{
				Optional:    true,
				Description: "The API token to use for authentication. For more details, refer to the documentation here infisical.com/docs/integrations/app-connections/1password",
				Sensitive:   true,
			},
		},
		ReadCredentialsForCreateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			credentialsConfig := make(map[string]interface{})

			var credentials AppConnection1PasswordCredentialsModel
			diags := plan.Credentials.As(ctx, &credentials, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != AppConnection1PasswordAuthMethodApiToken {
				diags.AddError(
					"Unable to create 1Password app connection",
					"Invalid method. Only api-token method is supported",
				)
				return nil, diags
			}

			if credentials.InstanceUrl.IsNull() || credentials.InstanceUrl.ValueString() == "" {
				diags.AddError(
					"Unable to create 1Password app connection",
					"instance_url field must be defined in api-token method",
				)
				return nil, diags
			}
			if credentials.ApiToken.IsNull() || credentials.ApiToken.ValueString() == "" {
				diags.AddError(
					"Unable to create 1Password app connection",
					"api_token field must be defined in api-token method",
				)
				return nil, diags
			}

			credentialsConfig["instanceUrl"] = credentials.InstanceUrl.ValueString()
			credentialsConfig["apiToken"] = credentials.ApiToken.ValueString()

			return credentialsConfig, diags
		},
		ReadCredentialsForUpdateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel, state AppConnectionBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			credentialsConfig := make(map[string]interface{})

			var credentialsFromPlan AppConnection1PasswordCredentialsModel
			diags := plan.Credentials.As(ctx, &credentialsFromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var credentialsFromState AppConnection1PasswordCredentialsModel
			diags = state.Credentials.As(ctx, &credentialsFromState, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != AppConnection1PasswordAuthMethodApiToken {
				diags.AddError(
					"Unable to update 1Password app connection",
					"Invalid method. Only api-token method is supported",
				)
				return nil, diags
			}

			if credentialsFromPlan.InstanceUrl.IsNull() || credentialsFromPlan.InstanceUrl.ValueString() == "" {
				diags.AddError(
					"Unable to update 1Password app connection",
					"instance_url field must be defined in api-token method",
				)
				return nil, diags
			}
			if credentialsFromPlan.ApiToken.IsNull() || credentialsFromPlan.ApiToken.ValueString() == "" {
				diags.AddError(
					"Unable to update 1Password app connection",
					"api_token field must be defined in api-token method",
				)
				return nil, diags
			}

			if credentialsFromState.InstanceUrl.ValueString() != credentialsFromPlan.InstanceUrl.ValueString() {
				credentialsConfig["instanceUrl"] = credentialsFromPlan.InstanceUrl.ValueString()
			}

			if credentialsFromState.ApiToken.ValueString() != credentialsFromPlan.ApiToken.ValueString() {
				credentialsConfig["apiToken"] = credentialsFromPlan.ApiToken.ValueString()
			}

			return credentialsConfig, diags
		},
		OverwriteCredentialsFields: func(state *AppConnectionBaseResourceModel) diag.Diagnostics {
			credentialsConfig := map[string]attr.Value{
				"instance_url": types.StringNull(),
				"api_token":    types.StringNull(),
			}

			var diags diag.Diagnostics
			state.Credentials, diags = types.ObjectValue(map[string]attr.Type{
				"instance_url": types.StringType,
				"api_token":    types.StringType,
			}, credentialsConfig)

			return diags
		},
	}
}
