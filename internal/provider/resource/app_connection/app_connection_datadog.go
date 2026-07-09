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

type AppConnectionDatadogCredentialsModel struct {
	Url            types.String `tfsdk:"url"`
	ApiKey         types.String `tfsdk:"api_key"`
	ApplicationKey types.String `tfsdk:"application_key"`
}

const AppConnectionDatadogAuthMethodApiKey = "api-key"

func NewAppConnectionDatadogResource() resource.Resource {
	return &AppConnectionBaseResource{
		App:               infisical.AppConnectionAppDatadog,
		AppConnectionName: "Datadog",
		ResourceTypeName:  "_app_connection_datadog",
		AllowedMethods:    []string{AppConnectionDatadogAuthMethodApiKey},
		CredentialsAttributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Required:    true,
				Description: "The Datadog API URL for your site (e.g. https://api.datadoghq.com). For more details, refer to the documentation here infisical.com/docs/integrations/app-connections/datadog",
			},
			"api_key": schema.StringAttribute{
				Required:    true,
				Description: "The Datadog API key used to authenticate requests. For more details, refer to the documentation here infisical.com/docs/integrations/app-connections/datadog",
				Sensitive:   true,
			},
			"application_key": schema.StringAttribute{
				Required:    true,
				Description: "The Datadog application key used to authenticate requests. For more details, refer to the documentation here infisical.com/docs/integrations/app-connections/datadog",
				Sensitive:   true,
			},
		},
		ReadCredentialsForCreateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			credentialsConfig := make(map[string]any)

			var credentials AppConnectionDatadogCredentialsModel
			diags := plan.Credentials.As(ctx, &credentials, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != AppConnectionDatadogAuthMethodApiKey {
				diags.AddError(
					"Unable to create Datadog app connection",
					"Invalid method. Only api-key method is supported",
				)
				return nil, diags
			}

			credentialsConfig["url"] = credentials.Url.ValueString()
			credentialsConfig["apiKey"] = credentials.ApiKey.ValueString()
			credentialsConfig["applicationKey"] = credentials.ApplicationKey.ValueString()

			return credentialsConfig, diags
		},
		ReadCredentialsForUpdateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel, state AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			credentialsConfig := make(map[string]any)

			var credentialsFromPlan AppConnectionDatadogCredentialsModel
			diags := plan.Credentials.As(ctx, &credentialsFromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var credentialsFromState AppConnectionDatadogCredentialsModel
			diags = state.Credentials.As(ctx, &credentialsFromState, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != AppConnectionDatadogAuthMethodApiKey {
				diags.AddError(
					"Unable to update Datadog app connection",
					"Invalid method. Only api-key method is supported",
				)
				return nil, diags
			}

			url := credentialsFromPlan.Url
			if credentialsFromPlan.Url.IsUnknown() {
				url = credentialsFromState.Url
			}
			if !url.IsNull() {
				credentialsConfig["url"] = url.ValueString()
			}

			apiKey := credentialsFromPlan.ApiKey
			if credentialsFromPlan.ApiKey.IsUnknown() {
				apiKey = credentialsFromState.ApiKey
			}
			if !apiKey.IsNull() {
				credentialsConfig["apiKey"] = apiKey.ValueString()
			}

			applicationKey := credentialsFromPlan.ApplicationKey
			if credentialsFromPlan.ApplicationKey.IsUnknown() {
				applicationKey = credentialsFromState.ApplicationKey
			}
			if !applicationKey.IsNull() {
				credentialsConfig["applicationKey"] = applicationKey.ValueString()
			}

			return credentialsConfig, diags
		},
		OverwriteCredentialsFields: func(state *AppConnectionBaseResourceModel) diag.Diagnostics {
			credentialsConfig := map[string]attr.Value{
				"url":             types.StringNull(),
				"api_key":         types.StringNull(),
				"application_key": types.StringNull(),
			}

			var diags diag.Diagnostics
			state.Credentials, diags = types.ObjectValue(map[string]attr.Type{
				"url":             types.StringType,
				"api_key":         types.StringType,
				"application_key": types.StringType,
			}, credentialsConfig)

			return diags
		},
	}
}
