package resource

import (
	"context"
	"regexp"
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

type AppConnectionADCSCredentialsModel struct {
	Host     types.String `tfsdk:"host"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

const ADCSAppConnectionUsernamePasswordMethod = "username-password"

// Mirrors the API, which trims each credential field and then requires 1 to 255 characters, so a
// value of only whitespace fails at plan instead of at apply.
func adcsCredentialLengthValidators() []validator.String {
	return []validator.String{
		stringvalidator.LengthBetween(1, 255),
		stringvalidator.RegexMatches(regexp.MustCompile(`\S`), "must contain a character other than whitespace"),
	}
}

func buildADCSCredentials(credentials AppConnectionADCSCredentialsModel, method string, errorSummary string) (map[string]any, diag.Diagnostics) {
	var diags diag.Diagnostics

	if method != ADCSAppConnectionUsernamePasswordMethod {
		diags.AddError(errorSummary, "Invalid method. Only username-password method is supported")
		return nil, diags
	}

	if credentials.Host.IsNull() || credentials.Host.ValueString() == "" {
		diags.AddError(errorSummary, "Host field must be defined in username-password method")
		return nil, diags
	}

	if credentials.Username.IsNull() || credentials.Username.ValueString() == "" {
		diags.AddError(errorSummary, "Username field must be defined in username-password method")
		return nil, diags
	}

	if credentials.Password.IsNull() || credentials.Password.ValueString() == "" {
		diags.AddError(errorSummary, "Password field must be defined in username-password method")
		return nil, diags
	}

	return map[string]any{
		"host":     credentials.Host.ValueString(),
		"username": credentials.Username.ValueString(),
		"password": credentials.Password.ValueString(),
	}, diags
}

func NewAppConnectionADCSResource() resource.Resource {
	return &AppConnectionBaseResource{
		App:               infisical.AppConnectionAppADCS,
		AppConnectionName: "Microsoft ADCS",
		ResourceTypeName:  "_app_connection_adcs",
		DocsURL:           "https://infisical.com/docs/integrations/app-connections/adcs",
		// AD CS speaks DCOM, which Infisical cannot reach directly, so the API rejects every
		// operation on a connection without a gateway.
		SupportsGateway: true,
		RequiresGateway: true,
		AllowedMethods:  []string{ADCSAppConnectionUsernamePasswordMethod},
		CredentialsAttributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Required:    true,
				Description: "The DNS name (FQDN) of the AD CS server, for example `ca01.corp.example.com`.",
				Validators:  adcsCredentialLengthValidators(),
			},
			"username": schema.StringAttribute{
				Required:    true,
				Description: "The Windows account used for enrollment, in the form `DOMAIN\\user` or `user@domain`.",
				Validators:  adcsCredentialLengthValidators(),
			},
			"password": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "The password of the Windows account.",
				Validators:  adcsCredentialLengthValidators(),
			},
		},
		ReadCredentialsForCreateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			var credentials AppConnectionADCSCredentialsModel
			diags := plan.Credentials.As(ctx, &credentials, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			return buildADCSCredentials(credentials, plan.Method.ValueString(), "Unable to create Microsoft ADCS app connection")
		},
		ReadCredentialsForUpdateFromPlan: func(ctx context.Context, plan AppConnectionBaseResourceModel, state AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
			var credentialsFromPlan AppConnectionADCSCredentialsModel
			diags := plan.Credentials.As(ctx, &credentialsFromPlan, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			var credentialsFromState AppConnectionADCSCredentialsModel
			diags = state.Credentials.As(ctx, &credentialsFromState, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			credentials := credentialsFromPlan
			if credentialsFromPlan.Host.IsUnknown() {
				credentials.Host = credentialsFromState.Host
			}
			if credentialsFromPlan.Username.IsUnknown() {
				credentials.Username = credentialsFromState.Username
			}
			if credentialsFromPlan.Password.IsUnknown() {
				credentials.Password = credentialsFromState.Password
			}

			return buildADCSCredentials(credentials, plan.Method.ValueString(), "Unable to update Microsoft ADCS app connection")
		},
		OverwriteCredentialsFields: func(state *AppConnectionBaseResourceModel) diag.Diagnostics {
			var diags diag.Diagnostics
			state.Credentials, diags = types.ObjectValue(map[string]attr.Type{
				"host":     types.StringType,
				"username": types.StringType,
				"password": types.StringType,
			}, map[string]attr.Value{
				"host":     types.StringNull(),
				"username": types.StringNull(),
				"password": types.StringNull(),
			})

			return diags
		},
	}
}
