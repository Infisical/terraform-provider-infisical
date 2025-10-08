package pkg

import (
	"context"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

type unorderedListModifier struct{}

func UnorderedList() planmodifier.List {
	return unorderedListModifier{}
}

func (m unorderedListModifier) Description(_ context.Context) string {
	return "Ignores order changes in list elements. Treats lists as unordered sets."
}

func (m unorderedListModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m unorderedListModifier) PlanModifyList(ctx context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	// if theres no state, don't modify
	if req.StateValue.IsNull() {
		return
	}

	// if there's no plan, don't modify
	if req.PlanValue.IsNull() {
		return
	}

	var stateList, planList []string

	req.StateValue.ElementsAs(ctx, &stateList, false)
	req.PlanValue.ElementsAs(ctx, &planList, false)

	// if they are the same length and contain the same elements, use state value
	if len(stateList) == len(planList) && hasSameElements(stateList, planList) {
		resp.PlanValue = req.StateValue
	}
}

func hasSameElements(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	aCopy := make([]string, len(a))
	bCopy := make([]string, len(b))
	copy(aCopy, a)
	copy(bCopy, b)

	sort.Strings(aCopy)
	sort.Strings(bCopy)

	for i := range aCopy {
		if aCopy[i] != bCopy[i] {
			return false
		}
	}
	return true
}
