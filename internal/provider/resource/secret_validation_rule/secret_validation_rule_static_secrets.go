package resource

import (
	"context"
	"maps"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// secretConstraintsMaxLength is the ceiling the API puts on a secret key or value.
const secretConstraintsMaxLength = 65536

// staticSecretsConstraintsModel is the constraints block of a static secrets rule. Both halves are
// optional on their own, but the API rejects a rule that constrains neither.
type staticSecretsConstraintsModel struct {
	KeyConstraints   types.Object `tfsdk:"key_constraints"`
	ValueConstraints types.Object `tfsdk:"value_constraints"`
}

// valueConstraintsModel is the shared string constraints plus the one field only a stored value
// can have.
type valueConstraintsModel struct {
	MinLength       types.Int64  `tfsdk:"min_length"`
	MaxLength       types.Int64  `tfsdk:"max_length"`
	RegexPattern    types.String `tfsdk:"regex_pattern"`
	RequiredPrefix  types.String `tfsdk:"required_prefix"`
	RequiredSuffix  types.String `tfsdk:"required_suffix"`
	ReusePrevention types.Object `tfsdk:"reuse_prevention"`
}

// reusePreventionModel groups the API's two uniqueness settings, which sit flat on
// valueConstraints as uniqueAcrossLastVersions and uniqueWithinScope.
type reusePreventionModel struct {
	PreviousVersions  types.Int64 `tfsdk:"previous_versions"`
	UniqueWithinScope types.Bool  `tfsdk:"unique_within_scope"`
}

var reusePreventionAttrTypes = map[string]attr.Type{
	"previous_versions":   types.Int64Type,
	"unique_within_scope": types.BoolType,
}

// valueConstraintsAttrTypes mirrors valueConstraintsModel, extending the shared string constraints
// rather than restating them.
var valueConstraintsAttrTypes = func() map[string]attr.Type {
	result := make(map[string]attr.Type, len(stringConstraintsAttrTypes)+1)
	maps.Copy(result, stringConstraintsAttrTypes)
	result["reuse_prevention"] = types.ObjectType{AttrTypes: reusePreventionAttrTypes}

	return result
}()

var staticSecretsConstraintsAttrTypes = map[string]attr.Type{
	"key_constraints":   types.ObjectType{AttrTypes: stringConstraintsAttrTypes},
	"value_constraints": types.ObjectType{AttrTypes: valueConstraintsAttrTypes},
}

func valueConstraintsAttributes() map[string]schema.Attribute {
	attributes := stringConstraintsAttributes("secret value", secretConstraintsMaxLength, "")
	attributes["reuse_prevention"] = schema.SingleNestedAttribute{
		Optional:    true,
		Description: "Rejects a value for repeating one already in use. Omit to allow any value the other constraints accept.",
		Attributes: map[string]schema.Attribute{
			"previous_versions": schema.Int64Attribute{
				Optional:    true,
				Description: "How many of the secret's own previous versions the new value must differ from. Between 1 and 25.",
				Validators: []validator.Int64{
					int64validator.Between(1, 25),
					// An empty block would send no uniqueness setting and read back as null.
					int64validator.AtLeastOneOf(path.MatchRelative().AtParent().AtName("unique_within_scope")),
				},
			},
			"unique_within_scope": schema.BoolAttribute{
				Optional:    true,
				Description: "Set to `true` to reject a value that another secret in the rule's scope already holds. Requires blind indexing on the project.",
			},
		},
	}

	return attributes
}

func NewSecretValidationRuleStaticSecretsResource() resource.Resource {
	return &SecretValidationRuleBaseResource{
		Type:             infisical.SecretValidationRuleTypeStaticSecrets,
		RuleName:         "Static Secrets",
		ResourceTypeName: "_secret_validation_rule_static_secrets",
		ConstraintsAttributes: map[string]schema.Attribute{
			"key_constraints": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Constraints enforced on the secret key when a secret is created or renamed. Omit to leave keys unconstrained.",
				Attributes:  stringConstraintsAttributes("secret key", secretConstraintsMaxLength, ""),
				Validators: []validator.Object{
					objectvalidator.AtLeastOneOf(
						path.MatchRelative().AtParent().AtName("key_constraints"),
						path.MatchRelative().AtParent().AtName("value_constraints"),
					),
				},
			},
			"value_constraints": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Constraints enforced on the secret value when a secret is created or updated. Omit to leave values unconstrained.",
				Attributes:  valueConstraintsAttributes(),
			},
		},

		ReadConstraintsFromPlan: func(ctx context.Context, plan SecretValidationRuleBaseResourceModel, isCreate bool) (map[string]interface{}, diag.Diagnostics) {
			var diags diag.Diagnostics

			var model staticSecretsConstraintsModel
			diags.Append(plan.Constraints.As(ctx, &model, basetypes.ObjectAsOptions{})...)
			if diags.HasError() {
				return nil, diags
			}

			key, keyDiags := stringConstraintsFromObject(ctx, model.KeyConstraints)
			diags.Append(keyDiags...)

			value, valueDiags := valueConstraintsFromObject(ctx, model.ValueConstraints)
			diags.Append(valueDiags...)

			if diags.HasError() {
				return nil, diags
			}

			constraints := make(map[string]any)
			putConstraintBlock(constraints, "keyConstraints", key, isCreate)
			putConstraintBlock(constraints, "valueConstraints", value, isCreate)

			return constraints, diags
		},

		ReadConstraintsFromApi: func(_ context.Context, rule infisical.SecretValidationRule) (types.Object, diag.Diagnostics) {
			var diags diag.Diagnostics

			key, keyDiags := stringConstraintsToObject(rule.KeyConstraints)
			diags.Append(keyDiags...)

			value, valueDiags := valueConstraintsToObject(rule.ValueConstraints)
			diags.Append(valueDiags...)

			if diags.HasError() {
				return types.ObjectNull(staticSecretsConstraintsAttrTypes), diags
			}

			object, objectDiags := types.ObjectValue(staticSecretsConstraintsAttrTypes, map[string]attr.Value{
				"key_constraints":   key,
				"value_constraints": value,
			})
			diags.Append(objectDiags...)

			return object, diags
		},
	}
}

// stringConstraintsToObject turns an API constraint block into state. A block the API did not set
// reads back as a null object, which is what an omitted block in the configuration looks like.
func stringConstraintsToObject(constraints *infisical.SecretValidationRuleStringConstraints) (types.Object, diag.Diagnostics) {
	if constraints == nil {
		return types.ObjectNull(stringConstraintsAttrTypes), nil
	}

	return types.ObjectValue(stringConstraintsAttrTypes, stringConstraintsValues(constraints))
}

func putConstraintBlock[T any](payload map[string]any, apiKey string, block *T, isCreate bool) {
	if block != nil {
		payload[apiKey] = block
		return
	}

	if !isCreate {
		payload[apiKey] = nil
	}
}

func valueConstraintsFromObject(ctx context.Context, object types.Object) (*infisical.SecretValidationRuleValueConstraints, diag.Diagnostics) {
	var diags diag.Diagnostics

	if object.IsNull() || object.IsUnknown() {
		return nil, diags
	}

	var model valueConstraintsModel
	diags.Append(object.As(ctx, &model, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	constraints := infisical.SecretValidationRuleValueConstraints{
		SecretValidationRuleStringConstraints: *stringConstraintsFromModel(stringConstraintsModel{
			MinLength:      model.MinLength,
			MaxLength:      model.MaxLength,
			RegexPattern:   model.RegexPattern,
			RequiredPrefix: model.RequiredPrefix,
			RequiredSuffix: model.RequiredSuffix,
		}),
	}

	if !(model.ReusePrevention.IsNull() || model.ReusePrevention.IsUnknown()) {
		var reusePrevention reusePreventionModel
		diags.Append(model.ReusePrevention.As(ctx, &reusePrevention, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}

		constraints.UniqueAcrossLastVersions = int64Pointer(reusePrevention.PreviousVersions)
		constraints.UniqueWithinScope = boolPointer(reusePrevention.UniqueWithinScope)
	}

	return &constraints, diags
}

func valueConstraintsToObject(constraints *infisical.SecretValidationRuleValueConstraints) (types.Object, diag.Diagnostics) {
	if constraints == nil {
		return types.ObjectNull(valueConstraintsAttrTypes), nil
	}

	values := stringConstraintsValues(&constraints.SecretValidationRuleStringConstraints)

	if constraints.UniqueAcrossLastVersions == nil && constraints.UniqueWithinScope == nil {
		values["reuse_prevention"] = types.ObjectNull(reusePreventionAttrTypes)
	} else {
		reusePrevention, diags := types.ObjectValue(reusePreventionAttrTypes, map[string]attr.Value{
			"previous_versions":   int64Value(constraints.UniqueAcrossLastVersions),
			"unique_within_scope": boolValue(constraints.UniqueWithinScope),
		})
		if diags.HasError() {
			return types.ObjectNull(valueConstraintsAttrTypes), diags
		}

		values["reuse_prevention"] = reusePrevention
	}

	return types.ObjectValue(valueConstraintsAttrTypes, values)
}
