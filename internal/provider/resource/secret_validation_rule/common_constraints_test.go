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
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func stringConstraintsObject(t *testing.T, constraints infisical.SecretValidationRuleStringConstraints) types.Object {
	t.Helper()

	object, diags := types.ObjectValue(stringConstraintsAttrTypes, stringConstraintsValues(&constraints))
	if diags.HasError() {
		t.Fatalf("building the string constraints object: %v", diags)
	}
	return object
}

func providerConstraintsObject(t *testing.T, providers []string, password types.Object) types.Object {
	t.Helper()

	providerSet, diags := types.SetValueFrom(context.Background(), types.StringType, providers)
	if diags.HasError() {
		t.Fatalf("building the providers set: %v", diags)
	}

	object, diags := types.ObjectValue(providerConstraintsAttrTypes, map[string]attr.Value{
		"providers":            providerSet,
		"password_constraints": password,
	})
	if diags.HasError() {
		t.Fatalf("building the provider constraints object: %v", diags)
	}
	return object
}

func int64Ptr(value int64) *int64    { return &value }
func stringPtr(value string) *string { return &value }
func boolPtr(value bool) *bool       { return &value }

// A block the configuration leaves out, or one that is not known until apply, sends nothing rather
// than a block of empty constraints.
func TestStringConstraintsFromObjectWithoutABlock(t *testing.T) {
	ctx := context.Background()

	for name, object := range map[string]types.Object{
		"null":    types.ObjectNull(stringConstraintsAttrTypes),
		"unknown": types.ObjectUnknown(stringConstraintsAttrTypes),
	} {
		t.Run(name, func(t *testing.T) {
			got, diags := stringConstraintsFromObject(ctx, object)
			if diags.HasError() {
				t.Fatalf("stringConstraintsFromObject() diagnostics = %v", diags)
			}
			if got != nil {
				t.Errorf("stringConstraintsFromObject() = %+v, want nil", got)
			}
		})
	}
}

// Every constraint the API returns has to come back out of state as the same constraint, or a rule
// read after an import would send something different on its next update.
func TestStringConstraintsRoundTrip(t *testing.T) {
	cases := map[string]infisical.SecretValidationRuleStringConstraints{
		"every field": {
			MinLength:      int64Ptr(8),
			MaxLength:      int64Ptr(64),
			RegexPattern:   stringPtr("^[A-Z_]+$"),
			RequiredPrefix: stringPtr("APP_"),
			RequiredSuffix: stringPtr("_KEY"),
		},
		"some fields": {MinLength: int64Ptr(1), RequiredSuffix: stringPtr("_KEY")},
		"no fields":   {},
	}

	for name, constraints := range cases {
		t.Run(name, func(t *testing.T) {
			got, diags := stringConstraintsFromObject(context.Background(), stringConstraintsObject(t, constraints))
			if diags.HasError() {
				t.Fatalf("stringConstraintsFromObject() diagnostics = %v", diags)
			}
			if !reflect.DeepEqual(*got, constraints) {
				t.Errorf("round trip = %+v, want %+v", *got, constraints)
			}
		})
	}
}

// The API accepts a minimum longer than the maximum, and the rule it stores can never be satisfied,
// so the pair is rejected at plan time. The check runs on every string constraints block.
func TestMaxLengthAtLeastMinLength(t *testing.T) {
	r := ruleResources(t)["dynamic_secrets"]
	s := ruleSchema(t, r)
	maxLength := path.Root("constraints").AtName("password_constraints").AtName("max_length")

	cases := map[string]struct {
		min, max  types.Int64
		wantError bool
	}{
		"max above min":  {min: types.Int64Value(8), max: types.Int64Value(32)},
		"max equals min": {min: types.Int64Value(16), max: types.Int64Value(16)},
		"no min":         {min: types.Int64Null(), max: types.Int64Value(4)},
		"max below min":  {min: types.Int64Value(32), max: types.Int64Value(8), wantError: true},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			password, diags := types.ObjectValue(stringConstraintsAttrTypes, map[string]attr.Value{
				"min_length":      c.min,
				"max_length":      c.max,
				"regex_pattern":   types.StringNull(),
				"required_prefix": types.StringNull(),
				"required_suffix": types.StringNull(),
			})
			if diags.HasError() {
				t.Fatalf("building the password constraints: %v", diags)
			}

			config := ruleConfig(t, s, ruleModel(providerConstraintsObject(t, []string{"sql-database"}, password)))
			if got := validateInt64(t, s, config, maxLength, c.max); got != c.wantError {
				t.Errorf("max_length %v with min_length %v rejected = %v, want %v", c.max, c.min, got, c.wantError)
			}
		})
	}
}

func TestEmptyPasswordConstraintsRejected(t *testing.T) {
	resources := ruleResources(t)
	passwordPath := path.Root("constraints").AtName("password_constraints")

	for _, name := range []string{"dynamic_secrets", "secret_rotations"} {
		t.Run(name, func(t *testing.T) {
			s := ruleSchema(t, resources[name])

			attribute, ok := ruleAttribute(t, s, passwordPath).(schema.SingleNestedAttribute)
			if !ok {
				t.Fatal("password_constraints is not a single nested attribute")
			}

			rejected := func(password types.Object) bool {
				resp := &validator.ObjectResponse{}
				for _, v := range attribute.Validators {
					v.ValidateObject(context.Background(), validator.ObjectRequest{
						Path:           passwordPath,
						PathExpression: passwordPath.Expression(),
						ConfigValue:    password,
						Config:         tfsdk.Config{Schema: s},
					}, resp)
				}
				return resp.Diagnostics.HasError()
			}

			if !rejected(stringConstraintsObject(t, infisical.SecretValidationRuleStringConstraints{})) {
				t.Error("an empty password_constraints block passed validation, want it rejected")
			}
			if rejected(stringConstraintsObject(t, infisical.SecretValidationRuleStringConstraints{MaxLength: int64Ptr(64)})) {
				t.Error("a password_constraints block with max_length was rejected, want it accepted")
			}
		})
	}
}

// The API runs the pattern with RE2, so a pattern Go cannot compile is one Infisical rejects too.
func TestRegexPatternValidation(t *testing.T) {
	s := ruleSchema(t, ruleResources(t)["static_secrets"])
	regexPattern := path.Root("constraints").AtName("key_constraints").AtName("regex_pattern")

	cases := map[string]bool{
		"^[A-Z_]+$":    false,
		"(":            true,
		"^(?=.*[A-Z])": true,
		"":             true,
	}

	for pattern, wantError := range cases {
		t.Run(pattern, func(t *testing.T) {
			if got := validateString(t, s, tfsdk.Config{Schema: s}, regexPattern, types.StringValue(pattern)); got != wantError {
				t.Errorf("regex_pattern %q rejected = %v, want %v", pattern, got, wantError)
			}
		})
	}
}

func TestReadProviderConstraintsFromPlan(t *testing.T) {
	password := infisical.SecretValidationRuleStringConstraints{MinLength: int64Ptr(16), RequiredPrefix: stringPtr("pw-")}
	plan := ruleModel(providerConstraintsObject(t, []string{"sql-database", "milvus"}, stringConstraintsObject(t, password)))

	got, diags := readProviderConstraintsFromPlan(context.Background(), plan, true)
	if diags.HasError() {
		t.Fatalf("readProviderConstraintsFromPlan() diagnostics = %v", diags)
	}

	providers, ok := got["providers"].([]string)
	if !ok {
		t.Fatalf("providers = %T, want a []string", got["providers"])
	}
	if len(providers) != 2 {
		t.Errorf("providers = %v, want sql-database and milvus", providers)
	}

	gotPassword, ok := got["passwordConstraints"].(*infisical.SecretValidationRuleStringConstraints)
	if !ok {
		t.Fatalf("passwordConstraints = %T, want a *SecretValidationRuleStringConstraints", got["passwordConstraints"])
	}
	if !reflect.DeepEqual(*gotPassword, password) {
		t.Errorf("passwordConstraints = %+v, want %+v", *gotPassword, password)
	}
}

// passwordConstraints is required by the API, so a block that is still unknown is sent as an empty
// one rather than being left out of the payload.
func TestReadProviderConstraintsFromPlanWithUnknownPassword(t *testing.T) {
	plan := ruleModel(providerConstraintsObject(t, []string{"sql-database"}, types.ObjectUnknown(stringConstraintsAttrTypes)))

	got, diags := readProviderConstraintsFromPlan(context.Background(), plan, true)
	if diags.HasError() {
		t.Fatalf("readProviderConstraintsFromPlan() diagnostics = %v", diags)
	}

	password, ok := got["passwordConstraints"].(*infisical.SecretValidationRuleStringConstraints)
	if !ok || password == nil {
		t.Fatalf("passwordConstraints = %#v, want an empty block", got["passwordConstraints"])
	}
	if !reflect.DeepEqual(*password, infisical.SecretValidationRuleStringConstraints{}) {
		t.Errorf("passwordConstraints = %+v, want an empty block", *password)
	}
}

func TestReadProviderConstraintsFromApi(t *testing.T) {
	ctx := context.Background()

	rule := infisical.SecretValidationRule{
		Providers:           []string{"postgres-credentials"},
		PasswordConstraints: &infisical.SecretValidationRuleStringConstraints{MaxLength: int64Ptr(64)},
	}

	object, diags := readProviderConstraintsFromApi(ctx, rule)
	if diags.HasError() {
		t.Fatalf("readProviderConstraintsFromApi() diagnostics = %v", diags)
	}

	var model providerConstraintsModel
	if diags := tfsdk.ValueAs(ctx, object, &model); diags.HasError() {
		t.Fatalf("reading the constraints object: %v", diags)
	}

	var providers []string
	model.Providers.ElementsAs(ctx, &providers, false)
	if len(providers) != 1 || providers[0] != "postgres-credentials" {
		t.Errorf("providers = %v, want only postgres-credentials", providers)
	}

	var password stringConstraintsModel
	if diags := tfsdk.ValueAs(ctx, model.PasswordConstraints, &password); diags.HasError() {
		t.Fatalf("reading the password constraints: %v", diags)
	}
	if got := password.MaxLength.ValueInt64(); got != 64 {
		t.Errorf("password max_length = %d, want 64", got)
	}
	if !password.MinLength.IsNull() {
		t.Errorf("password min_length = %v, want null", password.MinLength)
	}
}

// Both attributes are required in the schema, so reading either back as null would put a null into
// state for a required attribute and show as a diff on the first plan after an import.
func TestReadProviderConstraintsFromApiWithoutConstraints(t *testing.T) {
	object, diags := readProviderConstraintsFromApi(context.Background(), infisical.SecretValidationRule{})
	if diags.HasError() {
		t.Fatalf("readProviderConstraintsFromApi() diagnostics = %v", diags)
	}

	var model providerConstraintsModel
	if diags := tfsdk.ValueAs(context.Background(), object, &model); diags.HasError() {
		t.Fatalf("reading the constraints object: %v", diags)
	}

	if model.Providers.IsNull() {
		t.Error("providers is null, want an empty set")
	}
	if model.PasswordConstraints.IsNull() {
		t.Error("password_constraints is null, want an empty block")
	}
}

// Only the providers each rule type supports are accepted, and a rule has to name at least one or it
// constrains nothing.
func TestProvidersValidation(t *testing.T) {
	ctx := context.Background()

	supported := map[string][]string{
		"dynamic_secrets":  dynamicSecretsProviders,
		"secret_rotations": secretRotationsProviders,
	}
	resources := ruleResources(t)

	for name, providers := range supported {
		t.Run(name, func(t *testing.T) {
			s := ruleSchema(t, resources[name])
			providersPath := path.Root("constraints").AtName("providers")

			attribute, ok := ruleAttribute(t, s, providersPath).(schema.SetAttribute)
			if !ok {
				t.Fatal("providers is not a set attribute")
			}

			rejected := func(values []string) bool {
				set, diags := types.SetValueFrom(ctx, types.StringType, values)
				if diags.HasError() {
					t.Fatalf("building the providers set: %v", diags)
				}

				resp := &validator.SetResponse{}
				for _, v := range attribute.Validators {
					v.ValidateSet(ctx, validator.SetRequest{
						Path:           providersPath,
						PathExpression: providersPath.Expression(),
						ConfigValue:    set,
						Config:         tfsdk.Config{Schema: s},
					}, resp)
				}
				return resp.Diagnostics.HasError()
			}

			if rejected(providers) {
				t.Errorf("every supported provider %v was rejected, want them accepted", providers)
			}
			if !rejected([]string{}) {
				t.Error("an empty providers set passed validation, want it rejected")
			}
			if !rejected([]string{"not-a-provider"}) {
				t.Error("an unsupported provider passed validation, want it rejected")
			}
		})
	}

	// The two rule types share a constraints block, so a provider of one type must not leak into the
	// other's accepted set.
	for _, provider := range dynamicSecretsProviders {
		for _, other := range secretRotationsProviders {
			if provider == other {
				t.Errorf("provider %q is accepted by both dynamic secrets and secret rotations rules", provider)
			}
		}
	}
}

// A null or unknown value is absent from the payload, and an absent field reads back as null, so an
// unset constraint stays unset through a create and a refresh.
func TestPointerHelpers(t *testing.T) {
	if int64Pointer(types.Int64Null()) != nil || int64Pointer(types.Int64Unknown()) != nil {
		t.Error("int64Pointer() of a null or unknown value is not nil")
	}
	if got := int64Pointer(types.Int64Value(5)); got == nil || *got != 5 {
		t.Errorf("int64Pointer(5) = %v, want 5", got)
	}

	if stringPointer(types.StringNull()) != nil || stringPointer(types.StringUnknown()) != nil {
		t.Error("stringPointer() of a null or unknown value is not nil")
	}
	if got := stringPointer(types.StringValue("")); got == nil || *got != "" {
		t.Errorf("stringPointer(\"\") = %v, want a pointer to an empty string", got)
	}

	if boolPointer(types.BoolNull()) != nil || boolPointer(types.BoolUnknown()) != nil {
		t.Error("boolPointer() of a null or unknown value is not nil")
	}
	if got := boolPointer(types.BoolValue(false)); got == nil || *got {
		t.Errorf("boolPointer(false) = %v, want a pointer to false", got)
	}

	if !int64Value(nil).IsNull() || !stringValue(nil).IsNull() {
		t.Error("a nil pointer did not read back as null")
	}
	if got := int64Value(int64Ptr(5)); got.ValueInt64() != 5 {
		t.Errorf("int64Value(5) = %v, want 5", got)
	}
	if got := stringValue(stringPtr("APP_")); got.ValueString() != "APP_" {
		t.Errorf("stringValue(APP_) = %v, want APP_", got)
	}
}
