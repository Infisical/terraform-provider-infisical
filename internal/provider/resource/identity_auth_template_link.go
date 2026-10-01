package resource

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Plan modifiers for the identity auth attributes a linked auth template can supply. While
// template_id is set, such an attribute is computed: it holds the state value while nothing
// changes and becomes unknown on any change, since the template may have moved in the meantime.
// While template_id is unset, each modifier restores the attribute's behavior from before
// templates existed.

// An unknown template_id resolves at apply, so it counts as linked.
func isTemplateLinked(ctx context.Context, config tfsdk.Config, diagnostics *diag.Diagnostics) bool {
	var templateID types.String
	diagnostics.Append(config.GetAttribute(ctx, path.Root("template_id"), &templateID)...)
	return !templateID.IsNull()
}

// defaultUnlessTemplateLinked applies a static default only while no template is linked. A
// schema Default cannot do this: it is applied before plan modifiers run, and would fight the
// value the template supplies.
type defaultUnlessTemplateLinked struct {
	value string
}

func (m defaultUnlessTemplateLinked) Description(_ context.Context) string {
	return "Defaults to " + m.value + " unless an auth template is linked."
}

func (m defaultUnlessTemplateLinked) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m defaultUnlessTemplateLinked) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.ConfigValue.IsNull() || isTemplateLinked(ctx, req.Config, &resp.Diagnostics) {
		return
	}
	resp.PlanValue = types.StringValue(m.value)
}

// nullUnlessTemplateLinked keeps an unset attribute null while no template is linked, so it
// behaves as the plain optional attribute it was before templates could supply it.
type nullUnlessTemplateLinked struct{}

func (m nullUnlessTemplateLinked) Description(_ context.Context) string {
	return "Stays null when unset unless an auth template is linked."
}

func (m nullUnlessTemplateLinked) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m nullUnlessTemplateLinked) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.ConfigValue.IsNull() || isTemplateLinked(ctx, req.Config, &resp.Diagnostics) {
		return
	}
	resp.PlanValue = types.StringNull()
}

// useStateForUnknownUnlessTemplateLinked is UseStateForUnknown while no template is linked. A
// linked attribute must stay unknown on change, because keeping the state value would promise a
// result the template can overturn.
type useStateForUnknownUnlessTemplateLinked struct{}

func (m useStateForUnknownUnlessTemplateLinked) Description(_ context.Context) string {
	return "Keeps the prior state value unless an auth template is linked."
}

func (m useStateForUnknownUnlessTemplateLinked) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m useStateForUnknownUnlessTemplateLinked) linkedOrKnown(ctx context.Context, state tfsdk.State, config tfsdk.Config, planValueIsUnknown bool, configValueIsUnknown bool, diagnostics *diag.Diagnostics) bool {
	// Mirrors UseStateForUnknown: nothing to keep on create or destroy, and a known plan value or
	// an unknown configured value is left alone.
	if state.Raw.IsNull() || !planValueIsUnknown || configValueIsUnknown {
		return true
	}
	return isTemplateLinked(ctx, config, diagnostics)
}

func (m useStateForUnknownUnlessTemplateLinked) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.Plan.Raw.IsNull() || m.linkedOrKnown(ctx, req.State, req.Config, req.PlanValue.IsUnknown(), req.ConfigValue.IsUnknown(), &resp.Diagnostics) {
		return
	}
	resp.PlanValue = req.StateValue
}

func (m useStateForUnknownUnlessTemplateLinked) PlanModifyList(ctx context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if req.Plan.Raw.IsNull() || m.linkedOrKnown(ctx, req.State, req.Config, req.PlanValue.IsUnknown(), req.ConfigValue.IsUnknown(), &resp.Diagnostics) {
		return
	}
	resp.PlanValue = req.StateValue
}
