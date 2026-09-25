package validators

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"regexp/syntax"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// RegexPattern returns a validator that checks a string is a valid RE2 regular expression.
// Lookaround assertions, backreferences and malformed syntax are rejected.
func RegexPattern() validator.String {
	return regexPatternValidator{}
}

type regexPatternValidator struct{}

func (v regexPatternValidator) Description(_ context.Context) string {
	return "value must be a valid RE2 regular expression"
}

func (v regexPatternValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v regexPatternValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	_, err := regexp.Compile(req.ConfigValue.ValueString())
	if err == nil {
		return
	}

	// Other RE2 implementations accept \C (any byte) and \cX (control character), which Go's
	// regexp does not, so those escapes are not treated as errors.
	var syntaxErr *syntax.Error
	if errors.As(err, &syntaxErr) && syntaxErr.Code == syntax.ErrInvalidEscape &&
		(syntaxErr.Expr == `\C` || syntaxErr.Expr == `\c`) {
		return
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid regular expression",
		fmt.Sprintf("Must be a valid RE2 regular expression, which does not support lookaround assertions or backreferences: %s.", err),
	)
}
