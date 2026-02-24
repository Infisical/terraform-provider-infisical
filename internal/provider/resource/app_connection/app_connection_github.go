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

type AppConnectionGithubCredentialsModel struct {
	PersonalAccessToken types.String `tfsdk:"personal_access_token"`
	InstanceType        types.String `tfsdk:"instance_type"`
	Host                types.String `tfsdk:"host"`
}

const AppConnectionGithubAuthMethodPat = "pat"

// buildGithubCredentialsForCreate validates plan credentials and returns the API payload for create.
// Extracted for unit testing.
func buildGithubCredentialsForCreate(ctx context.Context, plan AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
	credentialsConfig := make(map[string]any)

	var credentials AppConnectionGithubCredentialsModel
	diags := plan.Credentials.As(ctx, &credentials, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, diags
	}

	if plan.Method.ValueString() != AppConnectionGithubAuthMethodPat {
		diags.AddError(
			"Unable to create GitHub app connection",
			"Invalid method. Only pat (Personal Access Token) method is supported. Note: GitHub App and OAuth connections must be created through the Infisical UI.",
		)
		return nil, diags
	}

	instanceType := "cloud"
	if !credentials.InstanceType.IsNull() && credentials.InstanceType.ValueString() != "" {
		instanceType = credentials.InstanceType.ValueString()
		if instanceType != "cloud" && instanceType != "server" {
			diags.AddError(
				"Unable to create GitHub app connection",
				fmt.Sprintf("Invalid instance_type. Only 'cloud' and 'server' are supported - got %s", instanceType),
			)
			return nil, diags
		}
	}

	if instanceType == "server" {
		if credentials.Host.IsNull() || credentials.Host.ValueString() == "" {
			diags.AddError(
				"Unable to create GitHub app connection",
				"host is required when instance_type is 'server' (GitHub Enterprise).",
			)
			return nil, diags
		}
		credentialsConfig["host"] = credentials.Host.ValueString()
	} else if !credentials.Host.IsNull() && credentials.Host.ValueString() != "" {
		credentialsConfig["host"] = credentials.Host.ValueString()
	}

	credentialsConfig["personalAccessToken"] = credentials.PersonalAccessToken.ValueString()
	credentialsConfig["instanceType"] = instanceType

	return credentialsConfig, diags
}

// buildGithubCredentialsForUpdate validates plan/state credentials and returns the API payload for update.
// Extracted for unit testing.
func buildGithubCredentialsForUpdate(ctx context.Context, plan AppConnectionBaseResourceModel, state AppConnectionBaseResourceModel) (map[string]any, diag.Diagnostics) {
	credentialsConfig := make(map[string]any)

	var credentialsFromPlan AppConnectionGithubCredentialsModel
	diags := plan.Credentials.As(ctx, &credentialsFromPlan, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, diags
	}

	var credentialsFromState AppConnectionGithubCredentialsModel
	diags = state.Credentials.As(ctx, &credentialsFromState, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, diags
	}

	if plan.Method.ValueString() != AppConnectionGithubAuthMethodPat {
		diags.AddError(
			"Unable to update GitHub app connection",
			"Invalid method. Only pat (Personal Access Token) method is supported. Note: GitHub App and OAuth connections must be created through the Infisical UI.",
		)
		return nil, diags
	}

	instanceType := credentialsFromPlan.InstanceType
	if credentialsFromPlan.InstanceType.IsUnknown() {
		instanceType = credentialsFromState.InstanceType
	}
	instanceTypeStr := "cloud"
	if !instanceType.IsNull() && instanceType.ValueString() != "" {
		instanceTypeStr = instanceType.ValueString()
		if instanceTypeStr != "cloud" && instanceTypeStr != "server" {
			diags.AddError(
				"Unable to update GitHub app connection",
				fmt.Sprintf("Invalid instance_type. Only 'cloud' and 'server' are supported - got %s", instanceTypeStr),
			)
			return nil, diags
		}
	}

	host := credentialsFromPlan.Host
	if credentialsFromPlan.Host.IsUnknown() {
		host = credentialsFromState.Host
	}
	if instanceTypeStr == "server" {
		if host.IsNull() || host.ValueString() == "" {
			diags.AddError(
				"Unable to update GitHub app connection",
				"host is required when instance_type is 'server' (GitHub Enterprise).",
			)
			return nil, diags
		}
		credentialsConfig["host"] = host.ValueString()
	} else if !host.IsNull() && host.ValueString() != "" {
		credentialsConfig["host"] = host.ValueString()
	}

	credentialsConfig["instanceType"] = instanceTypeStr

	personalAccessToken := credentialsFromPlan.PersonalAccessToken
	if credentialsFromPlan.PersonalAccessToken.IsUnknown() {
		personalAccessToken = credentialsFromState.PersonalAccessToken
	}
	if !personalAccessToken.IsNull() {
		credentialsConfig["personalAccessToken"] = personalAccessToken.ValueString()
	}

	return credentialsConfig, diags
}

func NewAppConnectionGithubResource() resource.Resource {
	return &AppConnectionBaseResource{
		App:               infisical.AppConnectionAppGithub,
		AppConnectionName: "GitHub",
		ResourceTypeName:  "_app_connection_github",
		AllowedMethods:    []string{AppConnectionGithubAuthMethodPat},
		CredentialsAttributes: map[string]schema.Attribute{
			"personal_access_token": schema.StringAttribute{
				Required:    true,
				Description: "The Personal Access Token used to access GitHub.",
				Sensitive:   true,
			},
			"instance_type": schema.StringAttribute{
				Optional:    true,
				Description: "The type of GitHub instance. Use 'cloud' for GitHub.com (default) or 'server' for GitHub Enterprise. When 'server', host is required.",
			},
			"host": schema.StringAttribute{
				Optional:    true,
				Description: "The hostname of your GitHub Enterprise instance. Required when instance_type is 'server'.",
			},
		},
		ReadCredentialsForCreateFromPlan: buildGithubCredentialsForCreate,
		ReadCredentialsForUpdateFromPlan: buildGithubCredentialsForUpdate,
		OverwriteCredentialsFields: func(state *AppConnectionBaseResourceModel) diag.Diagnostics {
			credentialsConfig := map[string]attr.Value{
				"personal_access_token": types.StringNull(),
				"instance_type":         types.StringNull(),
				"host":                  types.StringNull(),
			}

			var diags diag.Diagnostics
			state.Credentials, diags = types.ObjectValue(map[string]attr.Type{
				"personal_access_token": types.StringType,
				"instance_type":         types.StringType,
				"host":                  types.StringType,
			}, credentialsConfig)

			return diags
		},
	}
}
