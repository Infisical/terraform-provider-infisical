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

type AppConnectionGitlabCredentialsModel struct {
	AccessToken     types.String `tfsdk:"access_token"`
	InstanceUrl     types.String `tfsdk:"instance_url"`
	AccessTokenType types.String `tfsdk:"access_token_type"`
}

const AppConnectionGitlabAuthMethodAccessToken = "access-token"

func NewAppConnectionGitlabResource() resource.Resource {
	return &AppConnectionBaseResource{
		App:               infisical.AppConnectionAppGitlab,
		AppConnectionName: "GitLab",
		ResourceTypeName:  "_app_connection_gitlab",
		AllowedMethods:    []string{AppConnectionGitlabAuthMethodAccessToken},
		CredentialsAttributes: map[string]schema.Attribute{
			"access_token": schema.StringAttribute{
				Required:    true,
				Description: "The GitLab access token for authentication.",
				Sensitive:   true,
			},
			"instance_url": schema.StringAttribute{
				Optional:    true,
				Description: "The GitLab instance URL (e.g., https://gitlab.com for GitLab.com or your self-hosted GitLab URL).",
			},
			"access_token_type": schema.StringAttribute{
				Required:    true,
				Description: "The type of access token. Supported options: 'project' and 'personal'",
			},
		},
		ReadCredentialsForCreateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			credentialsConfig := make(map[string]interface{})

			var credentials AppConnectionGitlabCredentialsModel
			diags := plan.Credentials.As(ctx, &credentials, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != AppConnectionGitlabAuthMethodAccessToken {
				diags.AddError(
					"Unable to create GitLab app connection",
					"Invalid method. Only access-token method is supported. Note: GitLab OAuth connections must be created through the Infisical UI.",
				)
				return nil, diags
			}

			if credentials.AccessTokenType.ValueString() != "project" || credentials.AccessTokenType.ValueString() != "personal" {
				diags.AddError(
					"Unable to update GitLab app connection",
					"Invalid access_token_type. Only 'project' and 'personal' is supported.",
				)
				return nil, diags
			}

			credentialsConfig["accessToken"] = credentials.AccessToken.ValueString()
			credentialsConfig["instanceUrl"] = credentials.InstanceUrl.ValueString()
			credentialsConfig["accessTokenType"] = credentials.AccessTokenType.ValueString()

			return credentialsConfig, diags
		},
		ReadCredentialsForUpdateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel, state AppConnectionBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			credentialsConfig := make(map[string]interface{})

			var credentialsFromPlan AppConnectionGitlabCredentialsModel
			diags := plan.Credentials.As(ctx, &credentialsFromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var credentialsFromState AppConnectionGitlabCredentialsModel
			diags = state.Credentials.As(ctx, &credentialsFromState, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != AppConnectionGitlabAuthMethodAccessToken {
				diags.AddError(
					"Unable to update GitLab app connection",
					"Invalid method. Only access-token method is supported. Note: GitLab OAuth connections must be created through the Infisical UI.",
				)
				return nil, diags
			}

			if credentialsFromPlan.AccessTokenType.ValueString() != "project" || credentialsFromPlan.AccessTokenType.ValueString() != "personal" {
				diags.AddError(
					"Unable to update GitLab app connection",
					"Invalid access_token_type. Only 'project' and 'personal' is supported.",
				)
				return nil, diags
			}

			if credentialsFromState.AccessToken.ValueString() != credentialsFromPlan.AccessToken.ValueString() {
				credentialsConfig["accessToken"] = credentialsFromPlan.AccessToken.ValueString()
			}
			if credentialsFromState.InstanceUrl.ValueString() != credentialsFromPlan.InstanceUrl.ValueString() {
				credentialsConfig["instanceUrl"] = credentialsFromPlan.InstanceUrl.ValueString()
			}
			if credentialsFromState.AccessTokenType.ValueString() != credentialsFromPlan.AccessTokenType.ValueString() {
				credentialsConfig["accessTokenType"] = credentialsFromPlan.AccessTokenType.ValueString()
			}

			return credentialsConfig, diags
		},
		OverwriteCredentialsFields: func(state *AppConnectionBaseResourceModel) diag.Diagnostics {
			credentialsConfig := map[string]attr.Value{
				"access_token":      types.StringNull(),
				"instance_url":      types.StringNull(),
				"access_token_type": types.StringNull(),
			}

			var diags diag.Diagnostics
			state.Credentials, diags = types.ObjectValue(map[string]attr.Type{
				"access_token":      types.StringType,
				"instance_url":      types.StringType,
				"access_token_type": types.StringType,
			}, credentialsConfig)

			return diags
		},
	}
}
