package validators

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Each case records whether the pattern is valid RE2.
func TestRegexPattern(t *testing.T) {
	cases := []struct {
		pattern string
		valid   bool
	}{
		{`^[A-Z]+$`, true},
		{`\d{3}`, true},
		{`(?i)abc`, true},
		{`(?i:abc)`, true},
		{`(?s).`, true},
		{`(?U)a+`, true},
		{`(?m)^a$`, true},
		{`\p{L}+`, true},
		{`\pL`, true},
		{`\p{Greek}`, true},
		{`(?P<n>a)`, true},
		{`(?<name>a)`, true},
		{`[[:alpha:]]+`, true},
		{`\A`, true},
		{`\z`, true},
		{`\b\w+\b`, true},
		{`\x41`, true},
		{`\x{41}`, true},
		{`\0`, true},
		{`\/`, true},
		{`\-`, true},
		{`\Q.*\E`, true},
		{`[^\S\n]`, true},
		{`.*?`, true},
		{`a{1000}`, true},

		{`(?=.*[A-Z]).+`, false},
		{`(?!x).+`, false},
		{`(?<=a)b`, false},
		{`(a)\1`, false},
		{`\Z`, false},
		{`\e`, false},
		{`\C`, false},
		{`\cA`, false},
		{`a{1001}`, false},
		{`a++`, false},
		{`a*+`, false},
		{`[`, false},
		{`[a-`, false},
	}

	for _, c := range cases {
		t.Run(c.pattern, func(t *testing.T) {
			resp := &validator.StringResponse{}
			RegexPattern().ValidateString(context.Background(), validator.StringRequest{
				Path:        path.Root("regex_pattern"),
				ConfigValue: types.StringValue(c.pattern),
			}, resp)

			if accepted := !resp.Diagnostics.HasError(); accepted != c.valid {
				t.Errorf("pattern %q: accepted=%v, want %v", c.pattern, accepted, c.valid)
			}
		})
	}
}
