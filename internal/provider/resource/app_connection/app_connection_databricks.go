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

type AppConnectionDatabricksCredentialsModel struct {
	ClientId     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
	WorkspaceUrl types.String `tfsdk:"workspace_url"`
}

const AppConnectionDatabricksAuthMethodServicePrincipal = "service-principal"

func NewAppConnectionDatabricksResource() resource.Resource {
	return &AppConnectionBaseResource{
		App:               infisical.AppConnectionAppDatabricks,
		AppConnectionName: "Databricks",
		ResourceTypeName:  "_app_connection_databricks",
		AllowedMethods:    []string{AppConnectionDatabricksAuthMethodServicePrincipal},
		CredentialsAttributes: map[string]schema.Attribute{
			"client_id": schema.StringAttribute{
				Required:    true,
				Description: "The client ID of the Databricks service principal.",
			},
			"client_secret": schema.StringAttribute{
				Required:    true,
				Description: "The client secret of the Databricks service principal.",
				Sensitive:   true,
			},
			"workspace_url": schema.StringAttribute{
				Required:    true,
				Description: "The workspace URL of the Databricks instance.",
			},
		},
		ReadCredentialsForCreateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			credentialsConfig := make(map[string]any)

			var credentials AppConnectionDatabricksCredentialsModel
			diags := plan.Credentials.As(ctx, &credentials, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != AppConnectionDatabricksAuthMethodServicePrincipal {
				diags.AddError(
					"Unable to create Databricks app connection",
					"Invalid method. Only service-principal method is supported",
				)
				return nil, diags
			}

			credentialsConfig["clientId"] = credentials.ClientId.ValueString()
			credentialsConfig["clientSecret"] = credentials.ClientSecret.ValueString()
			credentialsConfig["workspaceUrl"] = credentials.WorkspaceUrl.ValueString()

			return credentialsConfig, diags
		},
		ReadCredentialsForUpdateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel, state AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			credentialsConfig := make(map[string]any)

			var credentialsFromPlan AppConnectionDatabricksCredentialsModel
			diags := plan.Credentials.As(ctx, &credentialsFromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var credentialsFromState AppConnectionDatabricksCredentialsModel
			diags = state.Credentials.As(ctx, &credentialsFromState, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != AppConnectionDatabricksAuthMethodServicePrincipal {
				diags.AddError(
					"Unable to update Databricks app connection",
					"Invalid method. Only service-principal method is supported",
				)
				return nil, diags
			}

			if credentialsFromState.ClientId.ValueString() != credentialsFromPlan.ClientId.ValueString() {
				credentialsConfig["clientId"] = credentialsFromPlan.ClientId.ValueString()
			}
			if credentialsFromState.ClientSecret.ValueString() != credentialsFromPlan.ClientSecret.ValueString() {
				credentialsConfig["clientSecret"] = credentialsFromPlan.ClientSecret.ValueString()
			}
			if credentialsFromState.WorkspaceUrl.ValueString() != credentialsFromPlan.WorkspaceUrl.ValueString() {
				credentialsConfig["workspaceUrl"] = credentialsFromPlan.WorkspaceUrl.ValueString()
			}

			return credentialsConfig, diags
		},
		OverwriteCredentialsFields: func(state *AppConnectionBaseResourceModel) diag.Diagnostics {
			credentialsConfig := map[string]attr.Value{
				"client_id":     types.StringNull(),
				"client_secret": types.StringNull(),
				"workspace_url": types.StringNull(),
			}

			var diags diag.Diagnostics
			state.Credentials, diags = types.ObjectValue(map[string]attr.Type{
				"client_id":     types.StringType,
				"client_secret": types.StringType,
				"workspace_url": types.StringType,
			}, credentialsConfig)

			return diags
		},
	}
}
