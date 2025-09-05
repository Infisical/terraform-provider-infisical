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

// AppConnectionGcpCredentialsModel describes the data source data model.
type AppConnectionGcpCredentialsModel struct {
	ServiceAccountEmail types.String `tfsdk:"service_account_email"`
}

func NewAppConnectionGcpResource() resource.Resource {
	return &AppConnectionBaseResource{
		App:               infisical.AppConnectionAppGCP,
		AppConnectionName: "GCP",
		ResourceTypeName:  "_app_connection_gcp",
		AllowedMethods:    []string{"service-account-impersonation"},
		CredentialsAttributes: map[string]schema.Attribute{
			"service_account_email": schema.StringAttribute{
				Optional:    true,
				Description: "The service account email to connect with GCP. The service account ID (the part of the email before '@') must be suffixed with the first two sections of your organization ID e.g. service-account-df92581a-0fe9@my-project.iam.gserviceaccount.com. For more details, refer to the documentation here https://infisical.com/docs/integrations/app-connections/gcp#configure-service-account-for-infisical",
				Sensitive:   true,
			},
		},
		ReadCredentialsForCreateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			credentialsConfig := make(map[string]any)

			var credentials AppConnectionGcpCredentialsModel
			diags := plan.Credentials.As(ctx, &credentials, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if credentials.ServiceAccountEmail.IsNull() || credentials.ServiceAccountEmail.ValueString() == "" {
				diags.AddError(
					"Unable to create GCP app connection",
					"Service account email field must be defined",
				)
				return nil, diags
			}

			credentialsConfig["serviceAccountEmail"] = credentials.ServiceAccountEmail.ValueString()

			return credentialsConfig, diags
		},
		ReadCredentialsForUpdateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel, state AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			credentialsConfig := make(map[string]any)

			var credentialsFromPlan AppConnectionGcpCredentialsModel
			diags := plan.Credentials.As(ctx, &credentialsFromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var credentialsFromState AppConnectionGcpCredentialsModel
			diags = state.Credentials.As(ctx, &credentialsFromState, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			serviceAccountEmail := credentialsFromPlan.ServiceAccountEmail
			if credentialsFromPlan.ServiceAccountEmail.IsUnknown() {
				serviceAccountEmail = credentialsFromState.ServiceAccountEmail
			}

			if serviceAccountEmail.IsNull() || serviceAccountEmail.ValueString() == "" {
				diags.AddError(
					"Unable to update GCP app connection",
					"Service account email field must be defined",
				)
				return nil, diags
			}

			if !serviceAccountEmail.IsNull() {
				credentialsConfig["serviceAccountEmail"] = serviceAccountEmail.ValueString()
			}

			return credentialsConfig, diags
		},
		OverwriteCredentialsFields: func(state *AppConnectionBaseResourceModel) diag.Diagnostics {
			credentialsConfig := map[string]attr.Value{
				"service_account_email": types.StringNull(),
			}

			var diags diag.Diagnostics
			state.Credentials, diags = types.ObjectValue(map[string]attr.Type{
				"service_account_email": types.StringType,
			}, credentialsConfig)

			return diags
		},
	}
}
