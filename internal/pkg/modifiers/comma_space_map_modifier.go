package pkg

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// CommaSpaceMapModifier ensures consistent formatting of comma-separated strings in map values.
type CommaSpaceMapModifier struct{}

func (m CommaSpaceMapModifier) Description(ctx context.Context) string {
	return "Ensures consistent formatting of comma-separated strings in map values with spaces after commas"
}

func (m CommaSpaceMapModifier) MarkdownDescription(ctx context.Context) string {
	return "Ensures consistent formatting of comma-separated strings in map values with spaces after commas"
}

func (m CommaSpaceMapModifier) PlanModifyMap(ctx context.Context, req planmodifier.MapRequest, resp *planmodifier.MapResponse) {
	if req.PlanValue.IsUnknown() || req.PlanValue.IsNull() {
		return
	}

	planElements := req.PlanValue.Elements()
	newElements := make(map[string]types.String)

	// Track config format for each key
	configFormats := make(map[string]bool)

	// Detect config format for each key
	if !req.ConfigValue.IsNull() {
		configElements := req.ConfigValue.Elements()
		for key, v := range configElements {
			if str, ok := v.(types.String); ok && !str.IsNull() {
				configFormats[key] = strings.Contains(str.ValueString(), ", ")
			}
		}
	}

	// Fallback to state value if config format not found
	if !req.StateValue.IsNull() {
		stateElements := req.StateValue.Elements()
		for key, v := range stateElements {
			if _, exists := configFormats[key]; !exists {
				if str, ok := v.(types.String); ok && !str.IsNull() {
					configFormats[key] = strings.Contains(str.ValueString(), ", ")
				}
			}
		}
	}

	for key, value := range planElements {
		strValue, ok := value.(types.String)
		if !ok {
			continue
		}

		if !strValue.IsNull() && !strValue.IsUnknown() {
			parts := strings.Split(strValue.ValueString(), ",")
			for i, part := range parts {
				parts[i] = strings.TrimSpace(part)
			}

			useSpaces, found := configFormats[key]
			if !found {
				useSpaces = false
			}

			var formattedValue string
			if useSpaces {
				formattedValue = strings.Join(parts, ", ")
			} else {
				formattedValue = strings.Join(parts, ",")
			}

			newElements[key] = types.StringValue(formattedValue)
		} else {
			newElements[key] = strValue
		}
	}

	newMapValue, diags := types.MapValueFrom(ctx, types.StringType, newElements)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.PlanValue = newMapValue
}

// CommaSpaceMap returns a new instance of CommaSpaceMapModifier.
func CommaSpaceMap() CommaSpaceMapModifier {
	return CommaSpaceMapModifier{}
}
