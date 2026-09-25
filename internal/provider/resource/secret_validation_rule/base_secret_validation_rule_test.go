package resource

import (
	"context"
	"testing"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// ruleResources is every secret validation rule resource the provider registers, so behavior they
// share through the base resource is checked on each of them rather than on one stand-in.
func ruleResources(t *testing.T) map[string]*SecretValidationRuleBaseResource {
	t.Helper()

	constructors := map[string]func() resource.Resource{
		"static_secrets":   NewSecretValidationRuleStaticSecretsResource,
		"dynamic_secrets":  NewSecretValidationRuleDynamicSecretsResource,
		"secret_rotations": NewSecretValidationRuleSecretRotationsResource,
	}

	resources := make(map[string]*SecretValidationRuleBaseResource, len(constructors))
	for name, constructor := range constructors {
		r, ok := constructor().(*SecretValidationRuleBaseResource)
		if !ok {
			t.Fatalf("%s constructor returned %T, want a *SecretValidationRuleBaseResource", name, constructor())
		}
		resources[name] = r
	}
	return resources
}

func ruleSchema(t *testing.T, r *SecretValidationRuleBaseResource) schema.Schema {
	t.Helper()

	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics = %v", resp.Diagnostics)
	}
	return resp.Schema
}

func ruleModel(constraints types.Object) SecretValidationRuleBaseResourceModel {
	return SecretValidationRuleBaseResourceModel{
		ID:          types.StringValue("00000000-0000-0000-0000-000000000001"),
		Name:        types.StringValue("Production keys"),
		Description: types.StringNull(),
		ProjectID:   types.StringValue("project-1"),
		Environment: types.StringValue("prod"),
		SecretPath:  types.StringValue("/apps/**"),
		IsActive:    types.BoolValue(true),
		Constraints: constraints,
	}
}

// ruleConfig renders a model the way Terraform holds a configuration, so validators that read
// sibling attributes see the same values they would at plan time.
func ruleConfig(t *testing.T, s schema.Schema, model SecretValidationRuleBaseResourceModel) tfsdk.Config {
	t.Helper()

	ctx := context.Background()

	var value attr.Value
	if diags := tfsdk.ValueFrom(ctx, model, s.Type(), &value); diags.HasError() {
		t.Fatalf("converting the model to its schema type: %v", diags)
	}

	raw, err := value.ToTerraformValue(ctx)
	if err != nil {
		t.Fatalf("converting the model to a raw value: %v", err)
	}
	return tfsdk.Config{Raw: raw, Schema: s}
}

func ruleAttribute(t *testing.T, s schema.Schema, p path.Path) schema.Attribute {
	t.Helper()

	attribute, diags := s.AttributeAtPath(context.Background(), p)
	if diags.HasError() {
		t.Fatalf("reading the attribute at %s: %v", p, diags)
	}
	return attribute
}

// validateString runs every validator on the string attribute at p, the way the framework does
// during validation, and reports whether any of them rejected the value.
func validateString(t *testing.T, s schema.Schema, config tfsdk.Config, p path.Path, value types.String) bool {
	t.Helper()

	attribute, ok := ruleAttribute(t, s, p).(schema.StringAttribute)
	if !ok {
		t.Fatalf("%s is not a string attribute", p)
	}

	resp := &validator.StringResponse{}
	for _, v := range attribute.Validators {
		v.ValidateString(context.Background(), validator.StringRequest{
			Path:           p,
			PathExpression: p.Expression(),
			ConfigValue:    value,
			Config:         config,
		}, resp)
	}
	return resp.Diagnostics.HasError()
}

func validateInt64(t *testing.T, s schema.Schema, config tfsdk.Config, p path.Path, value types.Int64) bool {
	t.Helper()

	attribute, ok := ruleAttribute(t, s, p).(schema.Int64Attribute)
	if !ok {
		t.Fatalf("%s is not an int64 attribute", p)
	}

	resp := &validator.Int64Response{}
	for _, v := range attribute.Validators {
		v.ValidateInt64(context.Background(), validator.Int64Request{
			Path:           p,
			PathExpression: p.Expression(),
			ConfigValue:    value,
			Config:         config,
		}, resp)
	}
	return resp.Diagnostics.HasError()
}

// The type-mismatch error on Read names the resource that manages the rule it found, so the table it
// reads from has to agree with the type name each resource actually registers under.
func TestSecretValidationRuleMetadata(t *testing.T) {
	for name, r := range ruleResources(t) {
		t.Run(name, func(t *testing.T) {
			resp := &resource.MetadataResponse{}
			r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "infisical"}, resp)

			want, ok := secretValidationRuleResourceTypeNames[r.Type]
			if !ok {
				t.Fatalf("rule type %q has no entry in secretValidationRuleResourceTypeNames", r.Type)
			}
			if resp.TypeName != want {
				t.Errorf("TypeName = %q, want %q", resp.TypeName, want)
			}
		})
	}
}

// The resource model is converted to and from the schema on every operation, and the constraints
// object comes from each type's own ReadConstraintsFromApi, so an attribute type that does not match
// the schema is a runtime error on first use. Round-tripping catches every mismatch at once.
func TestSecretValidationRuleResourceModelMatchesSchema(t *testing.T) {
	ctx := context.Background()

	minLength := int64(8)
	prefix := "APP_"
	rule := infisical.SecretValidationRule{
		KeyConstraints:      &infisical.SecretValidationRuleStringConstraints{RequiredPrefix: &prefix},
		ValueConstraints:    &infisical.SecretValidationRuleValueConstraints{SecretValidationRuleStringConstraints: infisical.SecretValidationRuleStringConstraints{MinLength: &minLength}},
		Providers:           []string{"sql-database"},
		PasswordConstraints: &infisical.SecretValidationRuleStringConstraints{MinLength: &minLength},
	}

	for name, r := range ruleResources(t) {
		t.Run(name, func(t *testing.T) {
			s := ruleSchema(t, r)

			constraints, diags := r.ReadConstraintsFromApi(ctx, rule)
			if diags.HasError() {
				t.Fatalf("ReadConstraintsFromApi() diagnostics = %v", diags)
			}

			var value attr.Value
			if diags := tfsdk.ValueFrom(ctx, ruleModel(constraints), s.Type(), &value); diags.HasError() {
				t.Fatalf("converting the model to its schema type: %v", diags)
			}

			var roundTripped SecretValidationRuleBaseResourceModel
			if diags := tfsdk.ValueAs(ctx, value, &roundTripped); diags.HasError() {
				t.Fatalf("converting the schema type back to the model: %v", diags)
			}

			if !roundTripped.Constraints.Equal(constraints) {
				t.Errorf("constraints did not survive the round trip: got %v, want %v", roundTripped.Constraints, constraints)
			}
		})
	}
}

func importState(t *testing.T, r *SecretValidationRuleBaseResource, id string) *resource.ImportStateResponse {
	t.Helper()

	s := ruleSchema(t, r)
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{Raw: tftypes.NewValue(s.Type().TerraformType(context.Background()), nil), Schema: s},
	}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: id}, resp)
	return resp
}

func TestSecretValidationRuleImportState(t *testing.T) {
	const id = "4b8d1c1e-9f0a-4e3b-8c2d-6a7e5f4d3c2b"

	for name, r := range ruleResources(t) {
		t.Run(name, func(t *testing.T) {
			resp := importState(t, r, id)
			if resp.Diagnostics.HasError() {
				t.Fatalf("ImportState() diagnostics = %v", resp.Diagnostics)
			}

			var got types.String
			if diags := resp.State.GetAttribute(context.Background(), path.Root("id"), &got); diags.HasError() {
				t.Fatalf("reading the imported id: %v", diags)
			}
			if got.ValueString() != id {
				t.Errorf("imported id = %q, want %q", got.ValueString(), id)
			}
		})
	}
}

// Rules are fetched by ID under a type-scoped path, so anything but a UUID is a 404 or a 400 from the
// API. Rejecting it up front gives the practitioner an error that says what was wrong.
func TestSecretValidationRuleImportStateRejectsNonUUID(t *testing.T) {
	for name, r := range ruleResources(t) {
		t.Run(name, func(t *testing.T) {
			resp := importState(t, r, "production-keys")
			if !resp.Diagnostics.HasError() {
				t.Fatal("ImportState() with a non-UUID ID: no error, want one")
			}
			if got := resp.Diagnostics.Errors()[0].Summary(); got != "Invalid import ID" {
				t.Errorf("error summary = %q, want %q", got, "Invalid import ID")
			}
		})
	}
}

func TestSecretPathValidation(t *testing.T) {
	s := ruleSchema(t, ruleResources(t)["static_secrets"])
	secretPath := path.Root("secret_path")

	cases := map[string]bool{
		"/":        false,
		"/apps":    false,
		"/apps/**": false,
		"apps":     true,
		"apps/**":  true,
		"":         true,
	}

	for value, wantError := range cases {
		t.Run(value, func(t *testing.T) {
			if got := validateString(t, s, tfsdk.Config{Schema: s}, secretPath, types.StringValue(value)); got != wantError {
				t.Errorf("secret_path %q rejected = %v, want %v", value, got, wantError)
			}
		})
	}
}

// Infisical stores an empty description as no description, so an empty string in the configuration
// would read back as null and diff on every plan. It has to be rejected rather than accepted.
func TestDescriptionRejectsEmptyString(t *testing.T) {
	s := ruleSchema(t, ruleResources(t)["static_secrets"])
	description := path.Root("description")

	if !validateString(t, s, tfsdk.Config{Schema: s}, description, types.StringValue("")) {
		t.Error("an empty description passed validation, want it rejected")
	}
	if validateString(t, s, tfsdk.Config{Schema: s}, description, types.StringValue("Keys used by production services")) {
		t.Error("a non-empty description was rejected, want it accepted")
	}
}

// A rule is enforced unless the configuration says otherwise, which is also what the API assumes for
// a rule it reads back without an isActive.
func TestIsActiveDefaultsToTrue(t *testing.T) {
	s := ruleSchema(t, ruleResources(t)["static_secrets"])

	isActive, ok := ruleAttribute(t, s, path.Root("is_active")).(schema.BoolAttribute)
	if !ok {
		t.Fatal("is_active is not a bool attribute")
	}
	if isActive.Default == nil {
		t.Fatal("is_active has no default")
	}

	resp := &defaults.BoolResponse{}
	isActive.Default.DefaultBool(context.Background(), defaults.BoolRequest{}, resp)
	if !resp.PlanValue.ValueBool() {
		t.Errorf("is_active default = %v, want true", resp.PlanValue)
	}
}

func TestSecretValidationRuleConfigure(t *testing.T) {
	r := ruleResources(t)["static_secrets"]

	// The framework calls Configure before the provider is configured, with no provider data yet.
	resp := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("Configure() without provider data: diagnostics = %v, want none", resp.Diagnostics)
	}

	resp = &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: "not a client"}, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("Configure() with provider data that is not a client: no error, want one")
	}

	client := &infisical.Client{}
	resp = &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: client}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Configure() with a client: diagnostics = %v", resp.Diagnostics)
	}
	if r.client != client {
		t.Error("Configure() did not keep the client it was given")
	}
}
