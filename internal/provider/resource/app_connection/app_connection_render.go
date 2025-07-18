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

// AppConnectionRenderCredentialsModel describes the data source data model.
type AppConnectionRenderCredentialsModel struct {
	ApiKey types.String `tfsdk:"api_key"`
}

const AppConnectionRenderAuthMethodApiKey = "api-key"

func NewAppConnectionRenderResource() resource.Resource {
	return &AppConnectionBaseResource{
		App:               infisical.AppConnectionAppRender,
		AppConnectionName: "Render",
		ResourceTypeName:  "_app_connection_render",
		AllowedMethods:    []string{AppConnectionRenderAuthMethodApiKey},
		CredentialsAttributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Required:    true,
				Description: "The API key to use for authentication. For more details, refer to the documentation here infisical.com/docs/integrations/app-connections/render",
				Sensitive:   true,
			},
		},
		ReadCredentialsForCreateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			credentialsConfig := make(map[string]interface{})

			var credentials AppConnectionRenderCredentialsModel
			diags := plan.Credentials.As(ctx, &credentials, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != AppConnectionRenderAuthMethodApiKey {
				diags.AddError(
					"Unable to create Render app connection",
					"Invalid method. Only api-key method is supported",
				)
				return nil, diags
			}

			credentialsConfig["apiKey"] = credentials.ApiKey.ValueString()

			return credentialsConfig, diags
		},
		ReadCredentialsForUpdateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel, state AppConnectionBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			credentialsConfig := make(map[string]interface{})

			var credentialsFromPlan AppConnectionRenderCredentialsModel
			diags := plan.Credentials.As(ctx, &credentialsFromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var credentialsFromState AppConnectionRenderCredentialsModel
			diags = state.Credentials.As(ctx, &credentialsFromState, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != AppConnectionRenderAuthMethodApiKey {
				diags.AddError(
					"Unable to update Render app connection",
					"Invalid method. Only api-key method is supported",
				)
				return nil, diags
			}

			if credentialsFromState.ApiKey.ValueString() != credentialsFromPlan.ApiKey.ValueString() {
				credentialsConfig["apiKey"] = credentialsFromPlan.ApiKey.ValueString()
			}

			return credentialsConfig, diags
		},
		OverwriteCredentialsFields: func(state *AppConnectionBaseResourceModel) diag.Diagnostics {
			credentialsConfig := map[string]attr.Value{
				"api_key": types.StringNull(),
			}

			var diags diag.Diagnostics
			state.Credentials, diags = types.ObjectValue(map[string]attr.Type{
				"api_key": types.StringType,
			}, credentialsConfig)

			return diags
		},
	}
}
