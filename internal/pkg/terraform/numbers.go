package terraform

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Int64PtrIfKnown returns a pointer to the int64 value when it is set (known and
// non-null), and nil when it is null or unknown.
//
// This lets request structs use `*int64` with `omitempty` so that:
//   - an unset field (unknown/null) is omitted, letting the API apply its default
//   - an explicit 0 is sent as 0, so callers can reset a value to zero
//
// Note: types.Int64.ValueInt64Pointer() cannot be used directly because it
// returns a pointer to 0 (not nil) for unknown values.
func Int64PtrIfKnown(v types.Int64) *int64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	value := v.ValueInt64()
	return &value
}
