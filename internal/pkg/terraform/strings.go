package terraform

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func StringListToGoStringSlice(ctx context.Context, diagnostics diag.Diagnostics, stringList basetypes.ListValue) []string {
	tfStringSlice := make([]types.String, 0, len(stringList.Elements()))
	if !stringList.IsNull() && !stringList.IsUnknown() {
		diags := stringList.ElementsAs(ctx, &tfStringSlice, false)
		diagnostics.Append(diags...)
		if diagnostics.HasError() {
			return nil
		}
	}

	output := make([]string, 0, len(tfStringSlice))
	for _, el := range tfStringSlice {
		output = append(output, el.ValueString())
	}
	return output
}

func IsAttrValueEmpty(value attr.Value) bool {
	if value.IsNull() || value.IsUnknown() {
		return true
	}

	switch v := value.(type) {
	case types.String:
		return v.ValueString() == ""
	case types.List:
		return len(v.Elements()) == 0
	case types.Set:
		return len(v.Elements()) == 0
	case types.Map:
		return len(v.Elements()) == 0
	default:
		return false
	}
}

// PreserveStringIfTrimmedEqual returns the configValue if both apiValue and configValue
// are equal after trimming whitespace. This is useful for preserving user formatting
// when the semantic content is the same.
// Returns the apiValue if configValue is null/unknown or if trimmed values differ.
func PreserveStringIfTrimmedEqual(apiValue string, configValue types.String) string {
	if !configValue.IsNull() && !configValue.IsUnknown() {
		if strings.TrimSpace(apiValue) == strings.TrimSpace(configValue.ValueString()) {
			return configValue.ValueString()
		}
	}
	return apiValue
}
