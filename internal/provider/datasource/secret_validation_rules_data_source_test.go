package datasource

import (
	"context"
	"testing"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func secretValidationRuleAttributes(t *testing.T, rule infisical.SecretValidationRule) map[string]types.Object {
	t.Helper()

	object, diags := secretValidationRuleObject(context.Background(), rule)
	if diags.HasError() {
		t.Fatalf("secretValidationRuleObject() diagnostics = %v", diags)
	}

	constraints, ok := object.Attributes()["constraints"].(types.Object)
	if !ok {
		t.Fatalf("constraints = %T, want an object", object.Attributes()["constraints"])
	}
	return map[string]types.Object{"rule": object, "constraints": constraints}
}

// The resource reads an empty description as no description, because the API hands back "" for one
// that was never set. The data source has to agree, or `description != null` means different things
// depending on where the rule was read from.
func TestSecretValidationRuleObjectDescription(t *testing.T) {
	empty, set := "", "Keys used by production services"

	cases := map[string]struct {
		description *string
		want        types.String
	}{
		"absent": {description: nil, want: types.StringNull()},
		"empty":  {description: &empty, want: types.StringNull()},
		"set":    {description: &set, want: types.StringValue(set)},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			rule := secretValidationRuleAttributes(t, infisical.SecretValidationRule{ID: "rule-1", Description: c.description})["rule"]
			if got := rule.Attributes()["description"]; !got.Equal(c.want) {
				t.Errorf("description = %v, want %v", got, c.want)
			}
		})
	}
}

// unique_within_scope is an opt-in flag, and the resource reads false back as null. The data source
// reports it the same way, which is also what its documentation promises.
func TestSecretValidationRuleObjectUniqueWithinScope(t *testing.T) {
	yes, no := true, false

	cases := map[string]struct {
		uniqueWithinScope *bool
		want              types.Bool
	}{
		"absent": {uniqueWithinScope: nil, want: types.BoolNull()},
		"false":  {uniqueWithinScope: &no, want: types.BoolNull()},
		"true":   {uniqueWithinScope: &yes, want: types.BoolValue(true)},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			constraints := secretValidationRuleAttributes(t, infisical.SecretValidationRule{
				ID:               "rule-1",
				Type:             infisical.SecretValidationRuleTypeStaticSecrets,
				ValueConstraints: &infisical.SecretValidationRuleValueConstraints{UniqueWithinScope: c.uniqueWithinScope},
			})["constraints"]

			value, ok := constraints.Attributes()["value_constraints"].(types.Object)
			if !ok {
				t.Fatalf("value_constraints = %T, want an object", constraints.Attributes()["value_constraints"])
			}
			if got := value.Attributes()["unique_within_scope"]; !got.Equal(c.want) {
				t.Errorf("unique_within_scope = %v, want %v", got, c.want)
			}
		})
	}
}
