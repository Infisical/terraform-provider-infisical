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

type AppConnectionSpaceliftCredentialsModel struct {
	ApiUrl       types.String `tfsdk:"api_url"`
	ApiKeyId     types.String `tfsdk:"api_key_id"`
	ApiKeySecret types.String `tfsdk:"api_key_secret"`
}

const AppConnectionSpaceliftAuthMethodApiKeySecret = "api-key-secret"

func readSpaceliftCredentialsFromPlan(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
	var credentials AppConnectionSpaceliftCredentialsModel
	diags := plan.Credentials.As(ctx, &credentials, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, diags
	}

	return map[string]any{
		"apiUrl":       credentials.ApiUrl.ValueString(),
		"apiKeyId":     credentials.ApiKeyId.ValueString(),
		"apiKeySecret": credentials.ApiKeySecret.ValueString(),
	}, diags
}

func NewAppConnectionSpaceliftResource() resource.Resource {
	return &AppConnectionBaseResource{
		App:               infisical.AppConnectionAppSpacelift,
		AppConnectionName: "Spacelift",
		ResourceTypeName:  "_app_connection_spacelift",
		AllowedMethods:    []string{AppConnectionSpaceliftAuthMethodApiKeySecret},
		CredentialsAttributes: map[string]schema.Attribute{
			"api_url": schema.StringAttribute{
				Required:    true,
				Description: "The Spacelift API URL to connect with (e.g., 'https://mycorp.app.spacelift.io').",
			},
			"api_key_id": schema.StringAttribute{
				Required:    true,
				Description: "The API Key ID used to authenticate with Spacelift.",
			},
			"api_key_secret": schema.StringAttribute{
				Required:    true,
				Description: "The API Key Secret used to authenticate with Spacelift.",
				Sensitive:   true,
			},
		},
		ReadCredentialsForCreateFromPlan: readSpaceliftCredentialsFromPlan,
		ReadCredentialsForUpdateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel, _ AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			return readSpaceliftCredentialsFromPlan(ctx, plan)
		},
		OverwriteCredentialsFields: func(state *AppConnectionBaseResourceModel) diag.Diagnostics {
			var diags diag.Diagnostics
			state.Credentials, diags = types.ObjectValue(map[string]attr.Type{
				"api_url":        types.StringType,
				"api_key_id":     types.StringType,
				"api_key_secret": types.StringType,
			}, map[string]attr.Value{
				"api_url":        types.StringNull(),
				"api_key_id":     types.StringNull(),
				"api_key_secret": types.StringNull(),
			})

			return diags
		},
	}
}
