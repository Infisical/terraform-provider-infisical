package validators

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestObjectAtLeastOneAttributeSet(t *testing.T) {
	attrTypes := map[string]attr.Type{
		"min_length":    types.Int64Type,
		"regex_pattern": types.StringType,
	}
	object := func(minLength types.Int64, regexPattern types.String) types.Object {
		return types.ObjectValueMust(attrTypes, map[string]attr.Value{
			"min_length":    minLength,
			"regex_pattern": regexPattern,
		})
	}

	cases := map[string]struct {
		value     types.Object
		wantError bool
	}{
		"one set":       {value: object(types.Int64Value(8), types.StringNull())},
		"all set":       {value: object(types.Int64Value(8), types.StringValue("^a$"))},
		"one unknown":   {value: object(types.Int64Unknown(), types.StringNull())},
		"null block":    {value: types.ObjectNull(attrTypes)},
		"unknown block": {value: types.ObjectUnknown(attrTypes)},
		"empty block":   {value: object(types.Int64Null(), types.StringNull()), wantError: true},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			resp := &validator.ObjectResponse{}
			ObjectAtLeastOneAttributeSet().ValidateObject(context.Background(), validator.ObjectRequest{
				Path:        path.Root("password_constraints"),
				ConfigValue: c.value,
			}, resp)

			if got := resp.Diagnostics.HasError(); got != c.wantError {
				t.Errorf("rejected = %v, want %v", got, c.wantError)
			}
		})
	}
}
