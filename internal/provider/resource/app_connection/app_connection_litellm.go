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

type AppConnectionLiteLLMCredentialsModel struct {
	InstanceUrl types.String `tfsdk:"instance_url"`
	ApiKey      types.String `tfsdk:"api_key"`
}

const AppConnectionLiteLLMAuthMethodApiKey = "api-key"

func NewAppConnectionLiteLLMResource() resource.Resource {
	return &AppConnectionBaseResource{
		App:               infisical.AppConnectionAppLiteLLM,
		AppConnectionName: "LiteLLM",
		ResourceTypeName:  "_app_connection_litellm",
		AllowedMethods:    []string{AppConnectionLiteLLMAuthMethodApiKey},
		CredentialsAttributes: map[string]schema.Attribute{
			"instance_url": schema.StringAttribute{
				Required:    true,
				Description: "The base URL of your LiteLLM instance (e.g. https://litellm.example.com). For more details, refer to the documentation here infisical.com/docs/integrations/app-connections/litellm",
			},
			"api_key": schema.StringAttribute{
				Required:    true,
				Description: "The LiteLLM API key used to authenticate with the LiteLLM instance. For more details, refer to the documentation here infisical.com/docs/integrations/app-connections/litellm",
				Sensitive:   true,
			},
		},
		ReadCredentialsForCreateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			credentialsConfig := make(map[string]any)

			var credentials AppConnectionLiteLLMCredentialsModel
			diags := plan.Credentials.As(ctx, &credentials, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != AppConnectionLiteLLMAuthMethodApiKey {
				diags.AddError(
					"Unable to create LiteLLM app connection",
					"Invalid method. Only api-key method is supported",
				)
				return nil, diags
			}

			credentialsConfig["instanceUrl"] = credentials.InstanceUrl.ValueString()
			credentialsConfig["apiKey"] = credentials.ApiKey.ValueString()

			return credentialsConfig, diags
		},
		ReadCredentialsForUpdateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel, state AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			credentialsConfig := make(map[string]any)

			var credentialsFromPlan AppConnectionLiteLLMCredentialsModel
			diags := plan.Credentials.As(ctx, &credentialsFromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var credentialsFromState AppConnectionLiteLLMCredentialsModel
			diags = state.Credentials.As(ctx, &credentialsFromState, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != AppConnectionLiteLLMAuthMethodApiKey {
				diags.AddError(
					"Unable to update LiteLLM app connection",
					"Invalid method. Only api-key method is supported",
				)
				return nil, diags
			}

			instanceUrl := credentialsFromPlan.InstanceUrl
			if credentialsFromPlan.InstanceUrl.IsUnknown() {
				instanceUrl = credentialsFromState.InstanceUrl
			}
			if !instanceUrl.IsNull() {
				credentialsConfig["instanceUrl"] = instanceUrl.ValueString()
			}

			apiKey := credentialsFromPlan.ApiKey
			if credentialsFromPlan.ApiKey.IsUnknown() {
				apiKey = credentialsFromState.ApiKey
			}
			if !apiKey.IsNull() {
				credentialsConfig["apiKey"] = apiKey.ValueString()
			}

			return credentialsConfig, diags
		},
		OverwriteCredentialsFields: func(state *AppConnectionBaseResourceModel) diag.Diagnostics {
			credentialsConfig := map[string]attr.Value{
				"instance_url": types.StringNull(),
				"api_key":      types.StringNull(),
			}

			var diags diag.Diagnostics
			state.Credentials, diags = types.ObjectValue(map[string]attr.Type{
				"instance_url": types.StringType,
				"api_key":      types.StringType,
			}, credentialsConfig)

			return diags
		},
	}
}
