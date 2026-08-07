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

type AppConnectionCircleCICredentialsModel struct {
	ApiToken types.String `tfsdk:"api_token"`
	Host     types.String `tfsdk:"host"`
}

const AppConnectionCircleCIAuthMethodApiToken = "api-token"

func NewAppConnectionCircleCIResource() resource.Resource {
	return &AppConnectionBaseResource{
		App:               infisical.AppConnectionAppCircleCI,
		AppConnectionName: "CircleCI",
		ResourceTypeName:  "_app_connection_circleci",
		AllowedMethods:    []string{AppConnectionCircleCIAuthMethodApiToken},
		CredentialsAttributes: map[string]schema.Attribute{
			"api_token": schema.StringAttribute{
				Required:    true,
				Description: "The CircleCI API token for authentication.",
				Sensitive:   true,
			},
			"host": schema.StringAttribute{
				Optional:    true,
				Description: "The CircleCI host to connect with, for self-hosted instances. (default: https://circleci.com)",
			},
		},
		ReadCredentialsForCreateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			credentialsConfig := make(map[string]any)

			var credentials AppConnectionCircleCICredentialsModel
			diags := plan.Credentials.As(ctx, &credentials, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != AppConnectionCircleCIAuthMethodApiToken {
				diags.AddError(
					"Unable to create CircleCI app connection",
					"Invalid method. Only api-token method is supported",
				)
				return nil, diags
			}

			credentialsConfig["apiToken"] = credentials.ApiToken.ValueString()

			if !credentials.Host.IsNull() {
				credentialsConfig["host"] = credentials.Host.ValueString()
			}

			return credentialsConfig, diags
		},
		ReadCredentialsForUpdateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel, state AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			credentialsConfig := make(map[string]any)

			var credentialsFromPlan AppConnectionCircleCICredentialsModel
			diags := plan.Credentials.As(ctx, &credentialsFromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var credentialsFromState AppConnectionCircleCICredentialsModel
			diags = state.Credentials.As(ctx, &credentialsFromState, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != AppConnectionCircleCIAuthMethodApiToken {
				diags.AddError(
					"Unable to update CircleCI app connection",
					"Invalid method. Only api-token method is supported",
				)
				return nil, diags
			}

			apiToken := credentialsFromPlan.ApiToken
			if credentialsFromPlan.ApiToken.IsUnknown() {
				apiToken = credentialsFromState.ApiToken
			}
			if !apiToken.IsNull() {
				credentialsConfig["apiToken"] = apiToken.ValueString()
			}

			host := credentialsFromPlan.Host
			if credentialsFromPlan.Host.IsUnknown() {
				host = credentialsFromState.Host
			}
			if !host.IsNull() {
				credentialsConfig["host"] = host.ValueString()
			}

			return credentialsConfig, diags
		},
		OverwriteCredentialsFields: func(state *AppConnectionBaseResourceModel) diag.Diagnostics {
			credentialsConfig := map[string]attr.Value{
				"api_token": types.StringNull(),
				"host":      types.StringNull(),
			}

			var diags diag.Diagnostics
			state.Credentials, diags = types.ObjectValue(map[string]attr.Type{
				"api_token": types.StringType,
				"host":      types.StringType,
			}, credentialsConfig)

			return diags
		},
	}
}
