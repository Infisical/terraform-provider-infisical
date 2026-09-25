package resource

import (
	"context"
	"fmt"
	"strings"

	infisical "terraform-provider-infisical/internal/client"
	"terraform-provider-infisical/internal/pkg/validators"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const passwordConstraintsMaxLength = 2048

type stringConstraintsModel struct {
	MinLength      types.Int64  `tfsdk:"min_length"`
	MaxLength      types.Int64  `tfsdk:"max_length"`
	RegexPattern   types.String `tfsdk:"regex_pattern"`
	RequiredPrefix types.String `tfsdk:"required_prefix"`
	RequiredSuffix types.String `tfsdk:"required_suffix"`
}

var stringConstraintsAttrTypes = map[string]attr.Type{
	"min_length":      types.Int64Type,
	"max_length":      types.Int64Type,
	"regex_pattern":   types.StringType,
	"required_prefix": types.StringType,
	"required_suffix": types.StringType,
}

const passwordRegexNote = " Setting this makes Infisical build the password from the pattern and ignore the length constraints on the same rule."

func stringConstraintsAttributes(subject string, maxLength int64, regexNote string) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"min_length": schema.Int64Attribute{
			Optional:    true,
			Description: fmt.Sprintf("The minimum number of characters the %s must contain.", subject),
			Validators:  []validator.Int64{int64validator.Between(1, maxLength)},
		},
		"max_length": schema.Int64Attribute{
			Optional:    true,
			Description: fmt.Sprintf("The maximum number of characters the %s may contain.", subject),
			Validators: []validator.Int64{
				int64validator.Between(1, maxLength),
				// The API accepts min > max and the resulting rule can never be satisfied, so the
				// pair is checked here where the answer is already known.
				int64validator.AtLeastSumOf(path.MatchRelative().AtParent().AtName("min_length")),
			},
		},
		"regex_pattern": schema.StringAttribute{
			Optional:    true,
			Description: fmt.Sprintf("A regular expression the %s must match.", subject) + regexNote,
			Validators: []validator.String{
				stringvalidator.LengthBetween(1, 4096),
				validators.RegexPattern(),
			},
		},
		"required_prefix": schema.StringAttribute{
			Optional:    true,
			Description: fmt.Sprintf("A string the %s must start with.", subject),
			Validators:  []validator.String{stringvalidator.LengthBetween(1, 1024)},
		},
		"required_suffix": schema.StringAttribute{
			Optional:    true,
			Description: fmt.Sprintf("A string the %s must end with.", subject),
			Validators:  []validator.String{stringvalidator.LengthBetween(1, 1024)},
		},
	}
}

func stringConstraintsFromObject(ctx context.Context, object types.Object) (*infisical.SecretValidationRuleStringConstraints, diag.Diagnostics) {
	var diags diag.Diagnostics

	if object.IsNull() || object.IsUnknown() {
		return nil, diags
	}

	var model stringConstraintsModel
	diags.Append(object.As(ctx, &model, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	return stringConstraintsFromModel(model), diags
}

func stringConstraintsFromModel(model stringConstraintsModel) *infisical.SecretValidationRuleStringConstraints {
	return &infisical.SecretValidationRuleStringConstraints{
		MinLength:      int64Pointer(model.MinLength),
		MaxLength:      int64Pointer(model.MaxLength),
		RegexPattern:   stringPointer(model.RegexPattern),
		RequiredPrefix: stringPointer(model.RequiredPrefix),
		RequiredSuffix: stringPointer(model.RequiredSuffix),
	}
}

func stringConstraintsValues(constraints *infisical.SecretValidationRuleStringConstraints) map[string]attr.Value {
	return map[string]attr.Value{
		"min_length":      int64Value(constraints.MinLength),
		"max_length":      int64Value(constraints.MaxLength),
		"regex_pattern":   stringValue(constraints.RegexPattern),
		"required_prefix": stringValue(constraints.RequiredPrefix),
		"required_suffix": stringValue(constraints.RequiredSuffix),
	}
}

// providerConstraintsModel is the constraints block shared by the dynamic secrets and secret
// rotations rule types, which differ only in which providers they accept.
type providerConstraintsModel struct {
	Providers           types.Set    `tfsdk:"providers"`
	PasswordConstraints types.Object `tfsdk:"password_constraints"`
}

var providerConstraintsAttrTypes = map[string]attr.Type{
	"providers":            types.SetType{ElemType: types.StringType},
	"password_constraints": types.ObjectType{AttrTypes: stringConstraintsAttrTypes},
}

// providerConstraintsAttributes builds the constraints block for a provider-scoped rule type.
// subject names the kind of provider and providers is the set of slugs the API accepts.
func providerConstraintsAttributes(subject string, providers []string) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"providers": schema.SetAttribute{
			Required:    true,
			ElementType: types.StringType,
			Description: fmt.Sprintf(
				"The %s providers this rule applies to. Supported: %s.",
				subject, strings.Join(providers, ", "),
			),
			Validators: []validator.Set{
				setvalidator.SizeAtLeast(1),
				setvalidator.ValueStringsAre(stringvalidator.OneOf(providers...)),
			},
		},
		"password_constraints": schema.SingleNestedAttribute{
			Required:    true,
			Description: "Constraints the generated password must satisfy. These replace any password requirements configured on the resource itself.",
			Attributes:  stringConstraintsAttributes("generated password", passwordConstraintsMaxLength, passwordRegexNote),
		},
	}
}

func readProviderConstraintsFromPlan(ctx context.Context, plan SecretValidationRuleBaseResourceModel, _ bool) (map[string]interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	var model providerConstraintsModel
	diags.Append(plan.Constraints.As(ctx, &model, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	providers := []string{}
	diags.Append(model.Providers.ElementsAs(ctx, &providers, false)...)
	if diags.HasError() {
		return nil, diags
	}

	password, passwordDiags := stringConstraintsFromObject(ctx, model.PasswordConstraints)
	diags.Append(passwordDiags...)
	if diags.HasError() {
		return nil, diags
	}

	if password == nil {
		// password is required, so this only happens when the whole block is still unknown.
		password = &infisical.SecretValidationRuleStringConstraints{}
	}

	return map[string]interface{}{
		"providers":           providers,
		"passwordConstraints": password,
	}, diags
}

func readProviderConstraintsFromApi(ctx context.Context, rule infisical.SecretValidationRule) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	ruleProviders := rule.Providers
	if ruleProviders == nil {
		ruleProviders = []string{}
	}

	providers, providerDiags := types.SetValueFrom(ctx, types.StringType, ruleProviders)
	diags.Append(providerDiags...)
	if diags.HasError() {
		return types.ObjectNull(providerConstraintsAttrTypes), diags
	}

	// password is required in the schema and passwordConstraints is in the API response's required
	// set, so the object is always materialized. Reading it back as null would put a null into
	// state for a required attribute and show as a diff on the first plan after an import.
	passwordConstraints := rule.PasswordConstraints
	if passwordConstraints == nil {
		passwordConstraints = &infisical.SecretValidationRuleStringConstraints{}
	}

	password, passwordDiags := types.ObjectValue(stringConstraintsAttrTypes, stringConstraintsValues(passwordConstraints))
	diags.Append(passwordDiags...)
	if diags.HasError() {
		return types.ObjectNull(providerConstraintsAttrTypes), diags
	}

	object, objectDiags := types.ObjectValue(providerConstraintsAttrTypes, map[string]attr.Value{
		"providers":            providers,
		"password_constraints": password,
	})
	diags.Append(objectDiags...)

	return object, diags
}

func int64Pointer(value types.Int64) *int64 {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	result := value.ValueInt64()

	return &result
}

func stringPointer(value types.String) *string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	result := value.ValueString()

	return &result
}

func boolPointer(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	result := value.ValueBool()

	return &result
}

func boolValue(value *bool) types.Bool {
	if value == nil {
		return types.BoolNull()
	}

	return types.BoolValue(*value)
}

func int64Value(value *int64) types.Int64 {
	if value == nil {
		return types.Int64Null()
	}

	return types.Int64Value(*value)
}

func stringValue(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}

	return types.StringValue(*value)
}
