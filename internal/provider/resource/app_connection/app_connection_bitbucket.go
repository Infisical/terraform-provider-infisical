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

type AppConnectionBitbucketCredentialsModel struct {
	Email    types.String `tfsdk:"email"`
	ApiToken types.String `tfsdk:"api_token"`
}

const AppConnectionBitbucketAuthMethodApiToken = "api-token"

func NewAppConnectionBitbucketResource() resource.Resource {
	return &AppConnectionBaseResource{
		App:               infisical.AppConnectionAppBitbucket,
		AppConnectionName: "Bitbucket",
		ResourceTypeName:  "_app_connection_bitbucket",
		AllowedMethods:    []string{AppConnectionBitbucketAuthMethodApiToken},
		CredentialsAttributes: map[string]schema.Attribute{
			"email": schema.StringAttribute{
				Required:    true,
				Description: "The email address associated with the Bitbucket API token.",
			},
			"api_token": schema.StringAttribute{
				Required:    true,
				Description: "The Bitbucket API token for authentication.",
				Sensitive:   true,
			},
		},
		ReadCredentialsForCreateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			credentialsConfig := make(map[string]interface{})

			var credentials AppConnectionBitbucketCredentialsModel
			diags := plan.Credentials.As(ctx, &credentials, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != AppConnectionBitbucketAuthMethodApiToken {
				diags.AddError(
					"Unable to create Bitbucket app connection",
					"Invalid method. Only api-token method is supported",
				)
				return nil, diags
			}

			credentialsConfig["email"] = credentials.Email.ValueString()
			credentialsConfig["apiToken"] = credentials.ApiToken.ValueString()

			return credentialsConfig, diags
		},
		ReadCredentialsForUpdateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel, state AppConnectionBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			credentialsConfig := make(map[string]interface{})

			var credentialsFromPlan AppConnectionBitbucketCredentialsModel
			diags := plan.Credentials.As(ctx, &credentialsFromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var credentialsFromState AppConnectionBitbucketCredentialsModel
			diags = state.Credentials.As(ctx, &credentialsFromState, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != AppConnectionBitbucketAuthMethodApiToken {
				diags.AddError(
					"Unable to update Bitbucket app connection",
					"Invalid method. Only api-token method is supported",
				)
				return nil, diags
			}

			if credentialsFromState.Email.ValueString() != credentialsFromPlan.Email.ValueString() {
				credentialsConfig["email"] = credentialsFromPlan.Email.ValueString()
			}
			if credentialsFromState.ApiToken.ValueString() != credentialsFromPlan.ApiToken.ValueString() {
				credentialsConfig["apiToken"] = credentialsFromPlan.ApiToken.ValueString()
			}

			return credentialsConfig, diags
		},
		OverwriteCredentialsFields: func(state *AppConnectionBaseResourceModel) diag.Diagnostics {
			credentialsConfig := map[string]attr.Value{
				"email":     types.StringNull(),
				"api_token": types.StringNull(),
			}

			var diags diag.Diagnostics
			state.Credentials, diags = types.ObjectValue(map[string]attr.Type{
				"email":     types.StringType,
				"api_token": types.StringType,
			}, credentialsConfig)

			return diags
		},
	}
}
