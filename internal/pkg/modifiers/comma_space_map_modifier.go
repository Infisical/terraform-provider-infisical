package pkg

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// CommaSpaceMapModifier ensures consistent formatting of comma-separated strings in map values
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

	for key, value := range planElements {
		strValue := value.(types.String)
		if !strValue.IsNull() && !strValue.IsUnknown() {
			parts := strings.Split(strValue.ValueString(), ",")

			// Trim spaces from each part and rejoin with ", "
			for i, part := range parts {
				parts[i] = strings.TrimSpace(part)
			}

			formattedValue := strings.Join(parts, ", ")

			newElements[key] = types.StringValue(formattedValue)
		} else {
			// Preserve null/unknown values
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

func CommaSpaceMap() CommaSpaceMapModifier {
	return CommaSpaceMapModifier{}
}
