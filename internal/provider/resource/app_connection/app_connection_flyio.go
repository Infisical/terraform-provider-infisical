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

type AppConnectionFlyioCredentialsModel struct {
	AccessToken types.String `tfsdk:"access_token"`
}

const AppConnectionFlyioAuthMethodAccessToken = "access-token"

func NewAppConnectionFlyioResource() resource.Resource {
	return &AppConnectionBaseResource{
		App:               infisical.AppConnectionAppFlyio,
		AppConnectionName: "Fly.io",
		ResourceTypeName:  "_app_connection_flyio",
		AllowedMethods:    []string{AppConnectionFlyioAuthMethodAccessToken},
		CredentialsAttributes: map[string]schema.Attribute{
			"access_token": schema.StringAttribute{
				Required:    true,
				Description: "The Fly.io access token for authentication.",
				Sensitive:   true,
			},
		},
		ReadCredentialsForCreateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			credentialsConfig := make(map[string]any)

			var credentials AppConnectionFlyioCredentialsModel
			diags := plan.Credentials.As(ctx, &credentials, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != AppConnectionFlyioAuthMethodAccessToken {
				diags.AddError(
					"Unable to create Fly.io app connection",
					"Invalid method. Only access-token method is supported",
				)
				return nil, diags
			}

			credentialsConfig["accessToken"] = credentials.AccessToken.ValueString()

			return credentialsConfig, diags
		},
		ReadCredentialsForUpdateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel, state AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			credentialsConfig := make(map[string]any)

			var credentialsFromPlan AppConnectionFlyioCredentialsModel
			diags := plan.Credentials.As(ctx, &credentialsFromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var credentialsFromState AppConnectionFlyioCredentialsModel
			diags = state.Credentials.As(ctx, &credentialsFromState, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != AppConnectionFlyioAuthMethodAccessToken {
				diags.AddError(
					"Unable to update Fly.io app connection",
					"Invalid method. Only access-token method is supported",
				)
				return nil, diags
			}

			accessToken := credentialsFromPlan.AccessToken
			if credentialsFromPlan.AccessToken.IsUnknown() {
				accessToken = credentialsFromState.AccessToken
			}
			if !accessToken.IsNull() {
				credentialsConfig["accessToken"] = accessToken.ValueString()
			}

			return credentialsConfig, diags
		},
		OverwriteCredentialsFields: func(state *AppConnectionBaseResourceModel) diag.Diagnostics {
			credentialsConfig := map[string]attr.Value{
				"access_token": types.StringNull(),
			}

			var diags diag.Diagnostics
			state.Credentials, diags = types.ObjectValue(map[string]attr.Type{
				"access_token": types.StringType,
			}, credentialsConfig)

			return diags
		},
	}
}
