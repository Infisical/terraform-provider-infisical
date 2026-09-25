package resource

import (
	"context"
	"reflect"
	"testing"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func valueConstraintsObject(t *testing.T, constraints *infisical.SecretValidationRuleValueConstraints) types.Object {
	t.Helper()

	object, diags := valueConstraintsToObject(constraints)
	if diags.HasError() {
		t.Fatalf("building the value constraints object: %v", diags)
	}
	return object
}

func staticSecretsConstraintsObject(t *testing.T, key, value types.Object) types.Object {
	t.Helper()

	object, diags := types.ObjectValue(staticSecretsConstraintsAttrTypes, map[string]attr.Value{
		"key_constraints":   key,
		"value_constraints": value,
	})
	if diags.HasError() {
		t.Fatalf("building the static secrets constraints object: %v", diags)
	}
	return object
}

func staticSecretsPayload(t *testing.T, constraints types.Object, isCreate bool) map[string]any {
	t.Helper()

	r := ruleResources(t)["static_secrets"]
	payload, diags := r.ReadConstraintsFromPlan(context.Background(), ruleModel(constraints), isCreate)
	if diags.HasError() {
		t.Fatalf("ReadConstraintsFromPlan() diagnostics = %v", diags)
	}
	return payload
}

// A block left out of the configuration is left out of the create, so the API applies no constraint
// to that half of the secret.
func TestStaticSecretsReadConstraintsFromPlanOnCreate(t *testing.T) {
	key := stringConstraintsObject(t, infisical.SecretValidationRuleStringConstraints{RequiredPrefix: stringPtr("APP_")})
	payload := staticSecretsPayload(t, staticSecretsConstraintsObject(t, key, types.ObjectNull(valueConstraintsAttrTypes)), true)

	gotKey, ok := payload["keyConstraints"].(*infisical.SecretValidationRuleStringConstraints)
	if !ok {
		t.Fatalf("keyConstraints = %T, want a *SecretValidationRuleStringConstraints", payload["keyConstraints"])
	}
	if gotKey.RequiredPrefix == nil || *gotKey.RequiredPrefix != "APP_" {
		t.Errorf("keyConstraints.requiredPrefix = %v, want APP_", gotKey.RequiredPrefix)
	}

	if _, present := payload["valueConstraints"]; present {
		t.Errorf("valueConstraints = %v, want it left out of the create", payload["valueConstraints"])
	}
}

// An update is a patch, so leaving a block out would keep whatever the rule had before. Removing a
// block from the configuration has to send an explicit null to clear it.
func TestStaticSecretsReadConstraintsFromPlanOnUpdate(t *testing.T) {
	key := stringConstraintsObject(t, infisical.SecretValidationRuleStringConstraints{RequiredPrefix: stringPtr("APP_")})
	payload := staticSecretsPayload(t, staticSecretsConstraintsObject(t, key, types.ObjectNull(valueConstraintsAttrTypes)), false)

	value, present := payload["valueConstraints"]
	if !present {
		t.Fatal("valueConstraints is missing from the update, want an explicit null to clear it")
	}
	if value != nil {
		t.Errorf("valueConstraints = %v, want nil", value)
	}
	if payload["keyConstraints"] == nil {
		t.Error("keyConstraints is nil, want the configured block")
	}
}

// reuse_prevention groups two settings the API keeps flat on valueConstraints, so they have to be
// spread back out under the API's own names.
func TestValueConstraintsFromObjectReusePrevention(t *testing.T) {
	ctx := context.Background()

	want := infisical.SecretValidationRuleValueConstraints{
		SecretValidationRuleStringConstraints: infisical.SecretValidationRuleStringConstraints{MinLength: int64Ptr(12)},
		UniqueAcrossLastVersions:              int64Ptr(5),
		UniqueWithinScope:                     boolPtr(true),
	}

	got, diags := valueConstraintsFromObject(ctx, valueConstraintsObject(t, &want))
	if diags.HasError() {
		t.Fatalf("valueConstraintsFromObject() diagnostics = %v", diags)
	}
	if !reflect.DeepEqual(*got, want) {
		t.Errorf("valueConstraintsFromObject() = %+v, want %+v", *got, want)
	}

	withoutReuse := infisical.SecretValidationRuleValueConstraints{
		SecretValidationRuleStringConstraints: infisical.SecretValidationRuleStringConstraints{MinLength: int64Ptr(12)},
	}
	got, diags = valueConstraintsFromObject(ctx, valueConstraintsObject(t, &withoutReuse))
	if diags.HasError() {
		t.Fatalf("valueConstraintsFromObject() diagnostics = %v", diags)
	}
	if got.UniqueAcrossLastVersions != nil || got.UniqueWithinScope != nil {
		t.Errorf("uniqueness settings = %v, %v, want both unset without a reuse_prevention block", got.UniqueAcrossLastVersions, got.UniqueWithinScope)
	}
}

func TestValueConstraintsFromObjectWithoutABlock(t *testing.T) {
	for name, object := range map[string]types.Object{
		"null":    types.ObjectNull(valueConstraintsAttrTypes),
		"unknown": types.ObjectUnknown(valueConstraintsAttrTypes),
	} {
		t.Run(name, func(t *testing.T) {
			got, diags := valueConstraintsFromObject(context.Background(), object)
			if diags.HasError() {
				t.Fatalf("valueConstraintsFromObject() diagnostics = %v", diags)
			}
			if got != nil {
				t.Errorf("valueConstraintsFromObject() = %+v, want nil", got)
			}
		})
	}
}

// A block the configuration omits reads back as null, so a rule the API returns without it, or
// without either uniqueness setting, matches the configuration that created it.
func TestValueConstraintsToObject(t *testing.T) {
	ctx := context.Background()

	if object := valueConstraintsObject(t, nil); !object.IsNull() {
		t.Errorf("valueConstraintsToObject(nil) = %v, want null", object)
	}

	readReusePrevention := func(constraints infisical.SecretValidationRuleValueConstraints) types.Object {
		var model valueConstraintsModel
		if diags := valueConstraintsObject(t, &constraints).As(ctx, &model, basetypes.ObjectAsOptions{}); diags.HasError() {
			t.Fatalf("reading the value constraints: %v", diags)
		}
		return model.ReusePrevention
	}

	if reuse := readReusePrevention(infisical.SecretValidationRuleValueConstraints{}); !reuse.IsNull() {
		t.Errorf("reuse_prevention without uniqueness settings = %v, want null", reuse)
	}

	reuse := readReusePrevention(infisical.SecretValidationRuleValueConstraints{UniqueAcrossLastVersions: int64Ptr(3)})
	if reuse.IsNull() {
		t.Fatal("reuse_prevention is null, want the uniqueness setting the API returned")
	}

	var model reusePreventionModel
	if diags := reuse.As(ctx, &model, basetypes.ObjectAsOptions{}); diags.HasError() {
		t.Fatalf("reading reuse_prevention: %v", diags)
	}
	if got := model.PreviousVersions.ValueInt64(); got != 3 {
		t.Errorf("previous_versions = %d, want 3", got)
	}
	if !model.UniqueWithinScope.IsNull() {
		t.Errorf("unique_within_scope = %v, want null", model.UniqueWithinScope)
	}
}

// Reading a rule back and planning it again has to send the same payload, or every plan after an
// import would propose an update.
func TestStaticSecretsConstraintsRoundTrip(t *testing.T) {
	r := ruleResources(t)["static_secrets"]

	rule := infisical.SecretValidationRule{
		ValueConstraints: &infisical.SecretValidationRuleValueConstraints{
			SecretValidationRuleStringConstraints: infisical.SecretValidationRuleStringConstraints{
				MinLength:    int64Ptr(16),
				RegexPattern: stringPtr("^[a-z0-9]+$"),
			},
			UniqueWithinScope: boolPtr(true),
		},
	}

	constraints, diags := r.ReadConstraintsFromApi(context.Background(), rule)
	if diags.HasError() {
		t.Fatalf("ReadConstraintsFromApi() diagnostics = %v", diags)
	}

	var model staticSecretsConstraintsModel
	if diags := constraints.As(context.Background(), &model, basetypes.ObjectAsOptions{}); diags.HasError() {
		t.Fatalf("reading the constraints: %v", diags)
	}
	if !model.KeyConstraints.IsNull() {
		t.Errorf("key_constraints = %v, want null for a rule without key constraints", model.KeyConstraints)
	}
	if model.ValueConstraints.IsNull() {
		t.Fatal("value_constraints is null, want the value constraints the API returned")
	}

	payload := staticSecretsPayload(t, constraints, true)
	if _, present := payload["keyConstraints"]; present {
		t.Errorf("keyConstraints = %v, want it left out", payload["keyConstraints"])
	}
	if got, ok := payload["valueConstraints"].(*infisical.SecretValidationRuleValueConstraints); !ok || !reflect.DeepEqual(*got, *rule.ValueConstraints) {
		t.Errorf("valueConstraints = %+v, want %+v", payload["valueConstraints"], *rule.ValueConstraints)
	}
}

// The API rejects a static secrets rule that constrains neither the key nor the value, so the
// configuration has to set at least one of the two blocks.
func TestStaticSecretsRequiresKeyOrValueConstraints(t *testing.T) {
	ctx := context.Background()
	s := ruleSchema(t, ruleResources(t)["static_secrets"])
	keyConstraints := path.Root("constraints").AtName("key_constraints")

	attribute, ok := ruleAttribute(t, s, keyConstraints).(schema.SingleNestedAttribute)
	if !ok {
		t.Fatal("key_constraints is not a single nested attribute")
	}

	rejected := func(key, value types.Object) bool {
		config := ruleConfig(t, s, ruleModel(staticSecretsConstraintsObject(t, key, value)))

		resp := &validator.ObjectResponse{}
		for _, v := range attribute.Validators {
			v.ValidateObject(ctx, validator.ObjectRequest{
				Path:           keyConstraints,
				PathExpression: keyConstraints.Expression(),
				ConfigValue:    key,
				Config:         config,
			}, resp)
		}
		return resp.Diagnostics.HasError()
	}

	key := stringConstraintsObject(t, infisical.SecretValidationRuleStringConstraints{RequiredPrefix: stringPtr("APP_")})
	value := valueConstraintsObject(t, &infisical.SecretValidationRuleValueConstraints{
		SecretValidationRuleStringConstraints: infisical.SecretValidationRuleStringConstraints{MinLength: int64Ptr(8)},
	})

	if !rejected(types.ObjectNull(stringConstraintsAttrTypes), types.ObjectNull(valueConstraintsAttrTypes)) {
		t.Error("a rule with neither key_constraints nor value_constraints passed validation, want it rejected")
	}
	if rejected(key, types.ObjectNull(valueConstraintsAttrTypes)) {
		t.Error("a rule with only key_constraints was rejected, want it accepted")
	}
	if rejected(types.ObjectNull(stringConstraintsAttrTypes), value) {
		t.Error("a rule with only value_constraints was rejected, want it accepted")
	}
}

// An empty reuse_prevention block sends no uniqueness setting and reads back as null, so it would diff
// on every plan. It has to set at least one of the two, and previous_versions has to be in the range
// the API allows.
func TestReusePreventionValidation(t *testing.T) {
	s := ruleSchema(t, ruleResources(t)["static_secrets"])
	previousVersions := path.Root("constraints").AtName("value_constraints").AtName("reuse_prevention").AtName("previous_versions")

	withReuse := func(previous types.Int64, uniqueWithinScope types.Bool) bool {
		reuse, diags := types.ObjectValue(reusePreventionAttrTypes, map[string]attr.Value{
			"previous_versions":   previous,
			"unique_within_scope": uniqueWithinScope,
		})
		if diags.HasError() {
			t.Fatalf("building reuse_prevention: %v", diags)
		}

		values := stringConstraintsValues(&infisical.SecretValidationRuleStringConstraints{})
		values["reuse_prevention"] = reuse
		value, diags := types.ObjectValue(valueConstraintsAttrTypes, values)
		if diags.HasError() {
			t.Fatalf("building value_constraints: %v", diags)
		}

		config := ruleConfig(t, s, ruleModel(staticSecretsConstraintsObject(t, types.ObjectNull(stringConstraintsAttrTypes), value)))
		return validateInt64(t, s, config, previousVersions, previous)
	}

	cases := map[string]struct {
		previous          types.Int64
		uniqueWithinScope types.Bool
		wantError         bool
	}{
		"previous versions only":   {previous: types.Int64Value(5), uniqueWithinScope: types.BoolNull()},
		"unique within scope only": {previous: types.Int64Null(), uniqueWithinScope: types.BoolValue(true)},
		"both":                     {previous: types.Int64Value(25), uniqueWithinScope: types.BoolValue(false)},
		"neither":                  {previous: types.Int64Null(), uniqueWithinScope: types.BoolNull(), wantError: true},
		"zero previous versions":   {previous: types.Int64Value(0), uniqueWithinScope: types.BoolNull(), wantError: true},
		"too many versions":        {previous: types.Int64Value(26), uniqueWithinScope: types.BoolNull(), wantError: true},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if got := withReuse(c.previous, c.uniqueWithinScope); got != c.wantError {
				t.Errorf("reuse_prevention {previous_versions = %v, unique_within_scope = %v} rejected = %v, want %v", c.previous, c.uniqueWithinScope, got, c.wantError)
			}
		})
	}
}
