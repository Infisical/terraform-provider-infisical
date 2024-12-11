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

	// Check config format if available
	var configFormat bool // true = spaces, false = no spaces
	if !req.ConfigValue.IsNull() {
		configElements := req.ConfigValue.Elements()
		// Look at first value to determine format
		for _, v := range configElements {
			if str, ok := v.(types.String); ok && !str.IsNull() {
				configFormat = strings.Contains(str.ValueString(), ", ")
				break
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

			var formattedValue string
			if configFormat {
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
