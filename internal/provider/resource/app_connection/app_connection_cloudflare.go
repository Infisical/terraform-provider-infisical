package resource

import (
	"context"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// AppConnectionCloudflareCredentialsModel describes the data source data model.
type AppConnectionCloudflareCredentialsModel struct {
	AccountId types.String `tfsdk:"account_id"`
	ApiToken  types.String `tfsdk:"api_token"`
}

const CloudflareAppConnectionApiTokenMethod = "api-token"

func NewAppConnectionCloudflareResource() resource.Resource {
	return &AppConnectionBaseResource{
		App:               infisical.AppConnectionAppCloudflare,
		AppConnectionName: "Cloudflare",
		ResourceTypeName:  "_app_connection_cloudflare",
		AllowedMethods:    []string{CloudflareAppConnectionApiTokenMethod},
		CredentialsAttributes: map[string]schema.Attribute{
			"account_id": schema.StringAttribute{
				Required:    true,
				Description: "The Cloudflare Account ID. This can be found in the sidebar of your Cloudflare dashboard.",
				Sensitive:   true,
			},
			"api_token": schema.StringAttribute{
				Required:    true,
				Description: "The Cloudflare API token with the necessary permissions to manage Workers scripts. The token should have Zone:Zone:Read, Zone:Zone Settings:Read, and Zone:Zone:Edit permissions.",
				Sensitive:   true,
			},
		},
		ReadCredentialsForCreateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			credentialsConfig := make(map[string]interface{})

			var credentials AppConnectionCloudflareCredentialsModel
			diags := plan.Credentials.As(ctx, &credentials, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if credentials.AccountId.IsNull() || credentials.AccountId.ValueString() == "" {
				diags.AddError(
					"Unable to create Cloudflare app connection",
					"Account ID field must be defined",
				)
				return nil, diags
			}

			if credentials.ApiToken.IsNull() || credentials.ApiToken.ValueString() == "" {
				diags.AddError(
					"Unable to create Cloudflare app connection",
					"API token field must be defined",
				)
				return nil, diags
			}

			credentialsConfig["accountId"] = credentials.AccountId.ValueString()
			credentialsConfig["apiToken"] = credentials.ApiToken.ValueString()

			return credentialsConfig, diags
		},
		ReadCredentialsForUpdateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel, state AppConnectionBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			credentialsConfig := make(map[string]interface{})

			var credentialsFromPlan AppConnectionCloudflareCredentialsModel
			diags := plan.Credentials.As(ctx, &credentialsFromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var credentialsFromState AppConnectionCloudflareCredentialsModel
			diags = state.Credentials.As(ctx, &credentialsFromState, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if credentialsFromPlan.AccountId.IsUnknown() {
				credentialsConfig["accountId"] = credentialsFromState.AccountId.ValueString()
			} else {
				if credentialsFromPlan.AccountId.IsNull() || credentialsFromPlan.AccountId.ValueString() == "" {
					diags.AddError(
						"Unable to update Cloudflare app connection",
						"Account ID field must be defined",
					)
					return nil, diags
				}
				credentialsConfig["accountId"] = credentialsFromPlan.AccountId.ValueString()
			}

			if credentialsFromPlan.ApiToken.IsUnknown() {
				credentialsConfig["apiToken"] = credentialsFromState.ApiToken.ValueString()
			} else {
				if credentialsFromPlan.ApiToken.IsNull() || credentialsFromPlan.ApiToken.ValueString() == "" {
					diags.AddError(
						"Unable to update Cloudflare app connection",
						"API token field must be defined",
					)
					return nil, diags
				}
				credentialsConfig["apiToken"] = credentialsFromPlan.ApiToken.ValueString()
			}

			return credentialsConfig, diags
		},
		OverwriteCredentialsFields: func(state *AppConnectionBaseResourceModel) diag.Diagnostics {
			return nil
		},
	}
}