package pkg

import (
	"context"
	pkg "terraform-provider-infisical/internal/pkg/strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type JsonEquivalentModifier struct{}

func (j JsonEquivalentModifier) Description(ctx context.Context) string {
	return "Compares JSON strings regardless of key ordering"
}

func (j JsonEquivalentModifier) MarkdownDescription(ctx context.Context) string {
	return "Compares JSON strings regardless of key ordering"
}

func (j JsonEquivalentModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
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

	// Normalize both JSONs
	normalizedPlan, err := pkg.NormalizeJSON(planJSON)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error normalizing plan JSON",
			err.Error(),
		)
		return
	}

	normalizedState, err := pkg.NormalizeJSON(stateJSON)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error normalizing state JSON",
			err.Error(),
		)
		return
	}

	// If the normalized versions are equal, use the config version
	// This is crucial: we keep the format from the config file
	if normalizedPlan == normalizedState {
		// Keep the original config value, not the normalized one
		resp.PlanValue = types.StringValue(planJSON)
	}
}
