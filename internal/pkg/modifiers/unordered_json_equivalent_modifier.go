package pkg

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type UnorderedJsonEquivalentModifier struct{}

func (j UnorderedJsonEquivalentModifier) Description(ctx context.Context) string {
	return "Compares JSON strings regardless of key ordering and array element ordering"
}

func (j UnorderedJsonEquivalentModifier) MarkdownDescription(ctx context.Context) string {
	return "Compares JSON strings regardless of key ordering and array element ordering"
}

func (j UnorderedJsonEquivalentModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// If the value hasn't changed, don't modify the plan
	if req.PlanValue.IsNull() || req.StateValue.IsNull() {
		return
	}

	// Get the values
	planJSON := req.PlanValue.ValueString()
	stateJSON := req.StateValue.ValueString()

	// If they're exactly the same, no need to compare further
	if planJSON == stateJSON {
		return
	}

	equal, err := areEquivalentJSON(planJSON, stateJSON)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error comparing JSON equivalence",
			err.Error(),
		)
		return
	}

	if equal {
		resp.PlanValue = types.StringValue(stateJSON)
	}
}

// areEquivalentJSON checks if two JSON strings are equivalent,
// ignoring key ordering in objects and element ordering in arrays
func areEquivalentJSON(json1, json2 string) (bool, error) {
	var obj1, obj2 interface{}

	if err := json.Unmarshal([]byte(json1), &obj1); err != nil {
		return false, err
	}

	if err := json.Unmarshal([]byte(json2), &obj2); err != nil {
		return false, err
	}

	return deepEqual(obj1, obj2), nil
}

// deepEqual recursively compares two values, with special handling for arrays
func deepEqual(v1, v2 interface{}) bool {
	// Handle nil values
	if v1 == nil || v2 == nil {
		return v1 == v2
	}

	// Type switch for more efficient and clearer comparison
	switch val1 := v1.(type) {
	case map[string]interface{}:
		// For maps/objects, compare each key/value pair
		val2, ok := v2.(map[string]interface{})
		if !ok || len(val1) != len(val2) {
			return false
		}

		for k, v1Val := range val1 {
			v2Val, ok := val2[k]
			if !ok || !deepEqual(v1Val, v2Val) {
				return false
			}
		}
		return true

	case []interface{}:
		// For slices/arrays, compare as unordered collections
		val2, ok := v2.([]interface{})
		if !ok || len(val1) != len(val2) {
			return false
		}

		// If arrays are empty, they're equal
		if len(val1) == 0 {
			return true
		}

		// Create a copy of val2 that we'll remove items from as we match them
		remaining := make([]interface{}, len(val2))
		copy(remaining, val2)

		// For each item in val1, find a matching item in remaining
		for _, item1 := range val1 {
			found := false
			for i, item2 := range remaining {
				if deepEqual(item1, item2) {
					// Remove the matched item from remaining by replacing with the last
					// element and shrinking the slice
					remaining[i] = remaining[len(remaining)-1]
					remaining = remaining[:len(remaining)-1]
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true

	case string:
		// Direct string comparison
		val2, ok := v2.(string)
		return ok && val1 == val2

	case float64:
		// JSON numbers are parsed as float64
		val2, ok := v2.(float64)
		return ok && val1 == val2

	case bool:
		// Boolean comparison
		val2, ok := v2.(bool)
		return ok && val1 == val2

	default:
		// Fall back to reflect.DeepEqual for any other types
		return reflect.DeepEqual(v1, v2)
	}
}
