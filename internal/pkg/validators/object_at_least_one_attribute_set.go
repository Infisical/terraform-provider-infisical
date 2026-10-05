package validators

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// ObjectAtLeastOneAttributeSet returns a validator for a nested block whose attributes are all
// optional but that means nothing when every one of them is left out. objectvalidator.AtLeastOneOf
// counts the block itself, so it cannot catch an empty `{}`.
func ObjectAtLeastOneAttributeSet() validator.Object {
	return objectAtLeastOneAttributeSetValidator{}
}

type objectAtLeastOneAttributeSetValidator struct{}

func (v objectAtLeastOneAttributeSetValidator) Description(_ context.Context) string {
	return "at least one attribute must be set"
}

func (v objectAtLeastOneAttributeSetValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v objectAtLeastOneAttributeSetValidator) ValidateObject(_ context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	attributes := req.ConfigValue.Attributes()
	for _, value := range attributes {
		// An unknown attribute may still turn out to be set, so the block is not known to be empty.
		if !value.IsNull() {
			return
		}
	}

	names := make([]string, 0, len(attributes))
	for name := range attributes {
		names = append(names, name)
	}
	slices.Sort(names)

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Empty constraints block",
		fmt.Sprintf("Set at least one of %s. Infisical rejects a block with none of them.", strings.Join(names, ", ")),
	)
}
