package validators

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// BoolTrueOrNull returns a validator for an opt-in flag whose false value means the same as leaving
// it out. The provider reads false back as null so imported resources match a configuration that
// omits the flag, which means a configured false would show as a diff on every plan.
func BoolTrueOrNull() validator.Bool {
	return boolTrueOrNullValidator{}
}

type boolTrueOrNullValidator struct{}

func (v boolTrueOrNullValidator) Description(_ context.Context) string {
	return "value must be true when set"
}

func (v boolTrueOrNullValidator) MarkdownDescription(_ context.Context) string {
	return "value must be `true` when set"
}

func (v boolTrueOrNullValidator) ValidateBool(_ context.Context, req validator.BoolRequest, resp *validator.BoolResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || req.ConfigValue.ValueBool() {
		return
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid value",
		"Set this to true to turn it on, or omit it to leave it off. false is the same as omitting it.",
	)
}
