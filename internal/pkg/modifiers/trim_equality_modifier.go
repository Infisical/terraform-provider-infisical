package pkg

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

type TrimEqualityModifier struct{}

func (m TrimEqualityModifier) Description(ctx context.Context) string {
	return "Treats strings as equal if they are the same after trimming whitespace"
}

func (m TrimEqualityModifier) MarkdownDescription(ctx context.Context) string {
	return "Treats strings as equal if they are the same after trimming whitespace"
}

func (m TrimEqualityModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Skip if values are null/unknown
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || req.StateValue.IsNull() {
		return
	}

	configTrimmed := strings.TrimSpace(req.ConfigValue.ValueString())
	stateTrimmed := strings.TrimSpace(req.StateValue.ValueString())

	// If trimmed values are equal, keep the state value to avoid unnecessary updates
	if configTrimmed == stateTrimmed {
		resp.PlanValue = req.StateValue
	}
}
