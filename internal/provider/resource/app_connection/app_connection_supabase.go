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

type AppConnectionSupabaseCredentialsModel struct {
	AccessKey   types.String `tfsdk:"access_key"`
	InstanceUrl types.String `tfsdk:"instance_url"`
}

const AppConnectionSupabaseAuthMethodAccessToken = "access-token"

func NewAppConnectionSupabaseResource() resource.Resource {
	return &AppConnectionBaseResource{
		App:               infisical.AppConnectionAppSupabase,
		AppConnectionName: "Supabase",
		ResourceTypeName:  "_app_connection_supabase",
		AllowedMethods:    []string{AppConnectionSupabaseAuthMethodAccessToken},
		CredentialsAttributes: map[string]schema.Attribute{
			"access_key": schema.StringAttribute{
				Required:    true,
				Description: "The Supabase access key for authentication.",
				Sensitive:   true,
			},
			"instance_url": schema.StringAttribute{
				Optional:    true,
				Description: "The Supabase instance URL (e.g., https://your-domain.com).",
			},
		},
		ReadCredentialsForCreateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			credentialsConfig := make(map[string]interface{})

			var credentials AppConnectionSupabaseCredentialsModel
			diags := plan.Credentials.As(ctx, &credentials, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != AppConnectionSupabaseAuthMethodAccessToken {
				diags.AddError(
					"Unable to create Supabase app connection",
					"Invalid method. Only access-token method is supported",
				)
				return nil, diags
			}

			credentialsConfig["accessKey"] = credentials.AccessKey.ValueString()
			if !credentials.InstanceUrl.IsNull() && !credentials.InstanceUrl.IsUnknown() {
				credentialsConfig["instanceUrl"] = credentials.InstanceUrl.ValueString()
			}

			return credentialsConfig, diags
		},
		ReadCredentialsForUpdateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel, state AppConnectionBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			credentialsConfig := make(map[string]interface{})

			var credentialsFromPlan AppConnectionSupabaseCredentialsModel
			diags := plan.Credentials.As(ctx, &credentialsFromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var credentialsFromState AppConnectionSupabaseCredentialsModel
			diags = state.Credentials.As(ctx, &credentialsFromState, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != AppConnectionSupabaseAuthMethodAccessToken {
				diags.AddError(
					"Unable to update Supabase app connection",
					"Invalid method. Only access-token method is supported",
				)
				return nil, diags
			}

			credentialsConfig["accessKey"] = credentialsFromPlan.AccessKey.ValueString()
			if credentialsFromState.InstanceUrl.ValueString() != credentialsFromPlan.InstanceUrl.ValueString() {
				if !credentialsFromPlan.InstanceUrl.IsNull() && !credentialsFromPlan.InstanceUrl.IsUnknown() {
					credentialsConfig["instanceUrl"] = credentialsFromPlan.InstanceUrl.ValueString()
				}
			}

			return credentialsConfig, diags
		},
		OverwriteCredentialsFields: func(state *AppConnectionBaseResourceModel) diag.Diagnostics {
			credentialsConfig := map[string]attr.Value{
				"access_key":   types.StringNull(),
				"instance_url": types.StringNull(),
			}

			var diags diag.Diagnostics
			state.Credentials, diags = types.ObjectValue(map[string]attr.Type{
				"access_key":   types.StringType,
				"instance_url": types.StringType,
			}, credentialsConfig)

			return diags
		},
	}
}
