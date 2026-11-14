package resource

import (
	"context"
	"regexp"
	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type DynamicSecretOpenRouterConfigurationModel struct {
	ManagementKey          types.String `tfsdk:"management_key"`
	ManagementKeyWO        types.String `tfsdk:"management_key_wo"`
	ManagementKeyWOVersion types.Int64  `tfsdk:"management_key_wo_version"`
	WorkspaceId            types.String `tfsdk:"workspace_id"`
	Limit                  types.String `tfsdk:"limit"`
	LimitReset             types.String `tfsdk:"limit_reset"`
	IncludeByokInLimit     types.Bool   `tfsdk:"include_byok_in_limit"`
}

var dynamicSecretOpenRouterConfigurationAttrTypes = map[string]attr.Type{
	"management_key":            types.StringType,
	"management_key_wo":         types.StringType,
	"management_key_wo_version": types.Int64Type,
	"workspace_id":              types.StringType,
	"limit":                     types.StringType,
	"limit_reset":               types.StringType,
	"include_byok_in_limit":     types.BoolType,
}

func optionalStringFromApi(inputs map[string]interface{}, key string) types.String {
	value, ok := inputs[key].(string)
	if !ok || value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}

func NewDynamicSecretOpenRouterResource() resource.Resource {
	return &DynamicSecretBaseResource{
		Provider:          infisical.DynamicSecretProviderOpenRouter,
		ResourceTypeName:  "_dynamic_secret_openrouter",
		DynamicSecretName: "OpenRouter",
		ConfigurationAttributes: map[string]schema.Attribute{
			"management_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "The OpenRouter management API key used to create and delete the API keys issued for each lease. This value is stored in the Terraform state; use management_key_wo to keep it out of state. Exactly one of management_key or management_key_wo must be set. Infisical never returns this value, so changes made outside of Terraform are not detected. The key can't be changed while the dynamic secret has active leases, and a new key must belong to the same OpenRouter organization. Self-hosted instances need an Infisical release newer than v0.167.10.",
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(
						path.MatchRelative().AtParent().AtName("management_key_wo"),
					),
				},
			},
			"management_key_wo": schema.StringAttribute{
				Optional:    true,
				WriteOnly:   true,
				Sensitive:   true,
				Description: "The OpenRouter management API key (write-only). This value is never stored in the Terraform state and can accept ephemeral values. Because it is not stored, changes to it are not detected; increment management_key_wo_version to push a new value. Requires Terraform 1.11+.",
				Validators: []validator.String{
					stringvalidator.AlsoRequires(
						path.MatchRelative().AtParent().AtName("management_key_wo_version"),
					),
				},
			},
			"management_key_wo_version": schema.Int64Attribute{
				Optional:    true,
				Description: "The version of the management_key_wo value. Increment this to trigger an update of the management key.",
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
					int64validator.AlsoRequires(
						path.MatchRelative().AtParent().AtName("management_key_wo"),
					),
				},
			},
			"workspace_id": schema.StringAttribute{
				Optional:    true,
				Description: "The ID of the OpenRouter workspace to create keys in. Defaults to the management key's default workspace.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 100),
				},
			},
			"limit": schema.StringAttribute{
				Optional:    true,
				Description: "The spending limit in USD applied to each created API key, as a decimal string such as \"2.5\". Leave unset for no limit.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^\d+(\.\d+)?$`), "must be a non-negative number, such as 10 or 2.5"),
				},
			},
			"limit_reset": schema.StringAttribute{
				Optional:    true,
				Description: "How often the spending limit resets. Supported values are daily, weekly and monthly. Leave unset for a limit that never resets.",
				Validators: []validator.String{
					stringvalidator.OneOf("daily", "weekly", "monthly"),
				},
			},
			"include_byok_in_limit": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether bring-your-own-key usage counts toward the spending limit. Defaults to false.",
			},
		},

		ReadConfigurationFromPlan: func(ctx context.Context, plan DynamicSecretBaseResourceModel, config DynamicSecretBaseResourceModel) (map[string]interface{}, diag.Diagnostics) {
			var configuration DynamicSecretOpenRouterConfigurationModel
			diags := plan.Configuration.As(ctx, &configuration, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, diags
			}

			// Write-only values are stripped from the plan, so read them from config.
			var rawConfiguration DynamicSecretOpenRouterConfigurationModel
			diags.Append(config.Configuration.As(ctx, &rawConfiguration, basetypes.ObjectAsOptions{})...)
			if diags.HasError() {
				return nil, diags
			}

			managementKey := configuration.ManagementKey.ValueString()
			if !rawConfiguration.ManagementKeyWO.IsNull() {
				managementKey = rawConfiguration.ManagementKeyWO.ValueString()
			}

			// PATCH merges inputs into the stored ones, so an unset field is sent as null to clear it
			// rather than omitted, which would keep the stored value.
			return map[string]interface{}{
				"managementKey":      managementKey,
				"workspaceId":        configuration.WorkspaceId.ValueStringPointer(),
				"limit":              configuration.Limit.ValueStringPointer(),
				"limitReset":         configuration.LimitReset.ValueStringPointer(),
				"includeByokInLimit": configuration.IncludeByokInLimit.ValueBool(),
			}, diags
		},

		ReadConfigurationFromApi: func(ctx context.Context, dynamicSecret infisical.DynamicSecret, configState types.Object) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			var existingConfig DynamicSecretOpenRouterConfigurationModel
			if !configState.IsNull() && !configState.IsUnknown() {
				diags.Append(configState.As(ctx, &existingConfig, basetypes.ObjectAsOptions{})...)
				if diags.HasError() {
					return types.ObjectNull(dynamicSecretOpenRouterConfigurationAttrTypes), diags
				}
			}

			if dynamicSecret.Inputs == nil {
				diags.AddError(
					"Unable to read dynamic secret configuration",
					"Infisical didn't return the configuration of the dynamic secret. Make sure the machine identity has the Read Root Credential permission on dynamic secrets.",
				)
				return types.ObjectNull(dynamicSecretOpenRouterConfigurationAttrTypes), diags
			}

			includeByokInLimit, _ := dynamicSecret.Inputs["includeByokInLimit"].(bool)

			// Infisical never returns managementKey, so the value comes from the prior state.
			configuration := map[string]attr.Value{
				"management_key":            existingConfig.ManagementKey,
				"management_key_wo":         types.StringNull(),
				"management_key_wo_version": existingConfig.ManagementKeyWOVersion,
				"workspace_id":              optionalStringFromApi(dynamicSecret.Inputs, "workspaceId"),
				"limit":                     optionalStringFromApi(dynamicSecret.Inputs, "limit"),
				"limit_reset":               optionalStringFromApi(dynamicSecret.Inputs, "limitReset"),
				"include_byok_in_limit":     types.BoolValue(includeByokInLimit),
			}

			obj, objDiags := types.ObjectValue(dynamicSecretOpenRouterConfigurationAttrTypes, configuration)
			diags.Append(objDiags...)
			return obj, diags
		},
	}
}
