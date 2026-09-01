package resource

import (
	"context"
	"strings"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type AppConnectionDigiCertCredentialsModel struct {
	ApiKey types.String `tfsdk:"api_key"`
	Region types.String `tfsdk:"region"`
}

const AppConnectionDigiCertAuthMethodApiKey = "api-key"

var AppConnectionDigiCertRegions = []string{"us", "eu"}

func NewAppConnectionDigiCertResource() resource.Resource {
	return &AppConnectionBaseResource{
		App:               infisical.AppConnectionAppDigiCert,
		AppConnectionName: "DigiCert",
		ResourceTypeName:  "_app_connection_digicert",
		AllowedMethods:    []string{AppConnectionDigiCertAuthMethodApiKey},
		CredentialsAttributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Required:    true,
				Description: "The DigiCert CertCentral API key used to authenticate requests. For more details, refer to the documentation here infisical.com/docs/integrations/app-connections/digicert",
				Sensitive:   true,
			},
			"region": schema.StringAttribute{
				Required:    true,
				Description: "The DigiCert CertCentral region your account belongs to. Possible values are: " + strings.Join(AppConnectionDigiCertRegions, ", "),
				Validators: []validator.String{
					stringvalidator.OneOf(AppConnectionDigiCertRegions...),
				},
			},
		},
		ReadCredentialsForCreateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			credentialsConfig := make(map[string]any)

			var credentials AppConnectionDigiCertCredentialsModel
			diags := plan.Credentials.As(ctx, &credentials, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != AppConnectionDigiCertAuthMethodApiKey {
				diags.AddError(
					"Unable to create DigiCert app connection",
					"Invalid method. Only api-key method is supported",
				)
				return nil, diags
			}

			credentialsConfig["apiKey"] = credentials.ApiKey.ValueString()
			credentialsConfig["region"] = credentials.Region.ValueString()

			return credentialsConfig, diags
		},
		ReadCredentialsForUpdateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel, state AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			credentialsConfig := make(map[string]any)

			var credentialsFromPlan AppConnectionDigiCertCredentialsModel
			diags := plan.Credentials.As(ctx, &credentialsFromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var credentialsFromState AppConnectionDigiCertCredentialsModel
			diags = state.Credentials.As(ctx, &credentialsFromState, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			if plan.Method.ValueString() != AppConnectionDigiCertAuthMethodApiKey {
				diags.AddError(
					"Unable to update DigiCert app connection",
					"Invalid method. Only api-key method is supported",
				)
				return nil, diags
			}

			apiKey := credentialsFromPlan.ApiKey
			if credentialsFromPlan.ApiKey.IsUnknown() {
				apiKey = credentialsFromState.ApiKey
			}
			if !apiKey.IsNull() {
				credentialsConfig["apiKey"] = apiKey.ValueString()
			}

			region := credentialsFromPlan.Region
			if credentialsFromPlan.Region.IsUnknown() {
				region = credentialsFromState.Region
			}
			if !region.IsNull() {
				credentialsConfig["region"] = region.ValueString()
			}

			return credentialsConfig, diags
		},
		OverwriteCredentialsFields: func(state *AppConnectionBaseResourceModel) diag.Diagnostics {
			credentialsConfig := map[string]attr.Value{
				"api_key": types.StringNull(),
				"region":  types.StringNull(),
			}

			var diags diag.Diagnostics
			state.Credentials, diags = types.ObjectValue(map[string]attr.Type{
				"api_key": types.StringType,
				"region":  types.StringType,
			}, credentialsConfig)

			return diags
		},
	}
}
