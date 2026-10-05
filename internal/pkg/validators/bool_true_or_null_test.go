package validators

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBoolTrueOrNull(t *testing.T) {
	cases := map[string]struct {
		value     types.Bool
		wantError bool
	}{
		"true":    {value: types.BoolValue(true)},
		"null":    {value: types.BoolNull()},
		"unknown": {value: types.BoolUnknown()},
		"false":   {value: types.BoolValue(false), wantError: true},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			resp := &validator.BoolResponse{}
			BoolTrueOrNull().ValidateBool(context.Background(), validator.BoolRequest{
				Path:        path.Root("unique_within_scope"),
				ConfigValue: c.value,
			}, resp)

			if got := resp.Diagnostics.HasError(); got != c.wantError {
				t.Errorf("rejected = %v, want %v", got, c.wantError)
			}
		})
	}
}
