package resource

import (
	"context"
	"fmt"
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
				Description: "The Access Token used to access GitLab.",
				Sensitive:   true,
			},
			"instance_url": schema.StringAttribute{
				Optional:    true,
				Description: "The GitLab instance URL to connect with. (default: https://gitlab.com)",
			},
			"access_token_type": schema.StringAttribute{
				Required:    true,
				Description: "The type of token used to connect with GitLab. Supported options: 'project' and 'personal'",
			},
		},
		ReadCredentialsForCreateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			credentialsConfig := make(map[string]any)

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

			accessTokenType := credentials.AccessTokenType.ValueString()

			if accessTokenType != "project" && accessTokenType != "personal" {
				diags.AddError(
					"Unable to update GitLab app connection",
					fmt.Sprintf("Invalid access_token_type. Only 'project' and 'personal' is supported - got %s", accessTokenType),
				)
				return nil, diags
			}

			if !credentials.InstanceUrl.IsNull() {
				credentialsConfig["instanceUrl"] = credentials.InstanceUrl.ValueString()
			}

			credentialsConfig["accessToken"] = credentials.AccessToken.ValueString()
			credentialsConfig["accessTokenType"] = accessTokenType

			return credentialsConfig, diags
		},
		ReadCredentialsForUpdateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel, state AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			credentialsConfig := make(map[string]any)

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

			accessTokenType := credentialsFromPlan.AccessTokenType
			if credentialsFromPlan.AccessTokenType.IsUnknown() {
				accessTokenType = credentialsFromState.AccessTokenType
			}

			if accessTokenType.IsNull() || (accessTokenType.ValueString() != "project" && accessTokenType.ValueString() != "personal") {
				diags.AddError(
					"Unable to update GitLab app connection",
					fmt.Sprintf("Invalid access_token_type. Only 'project' and 'personal' is supported - got %s", accessTokenType),
				)
				return nil, diags
			}

			credentialsConfig["accessTokenType"] = accessTokenType.ValueString()

			instanceUrl := credentialsFromPlan.InstanceUrl
			if credentialsFromPlan.InstanceUrl.IsUnknown() {
				instanceUrl = credentialsFromState.InstanceUrl
			}
			if !instanceUrl.IsNull() {
				credentialsConfig["instanceUrl"] = instanceUrl.ValueString()
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
