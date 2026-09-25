package terraform

import (
	"context"
	"fmt"
	"net/url"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var SlugRegexValidator = stringvalidator.RegexMatches(
	regexp.MustCompile(`^[a-z0-9-]+$`),
	"invalid slug, slugs must be lowercase alphanumeric characters and hyphens only (example-slug-1)",
)

var JsonStringValidator = stringvalidator.RegexMatches(
	regexp.MustCompile(`^[\[{].*[\]}]$`),
	"must be a valid JSON string",
)

var HttpsUrlValidator = stringvalidator.RegexMatches(
	regexp.MustCompile(`^https://\S+$`),
	"must be a valid URL starting with https:// (example: https://example.com)",
)

// HttpsPreferredUrlValidator accepts http and https URLs, warning on http. For destinations where
// plaintext is a legitimate choice on a trusted network but must not be made silently.
var HttpsPreferredUrlValidator = httpsPreferredUrlValidator{}

type httpsPreferredUrlValidator struct{}

func (v httpsPreferredUrlValidator) Description(_ context.Context) string {
	return "must be a valid http:// or https:// URL; http is accepted but warns, as traffic is unencrypted"
}

func (v httpsPreferredUrlValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v httpsPreferredUrlValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid URL",
			fmt.Sprintf("Expected a URL starting with https:// or http://, got: %s", value),
		)
		return
	}

	if parsed.Scheme == "http" {
		resp.Diagnostics.AddAttributeWarning(
			req.Path,
			"Unencrypted delivery",
			fmt.Sprintf(
				"%s uses http, so the payload and any headers authenticating to it are sent in the clear. "+
					"Use https unless the destination is only reachable over a trusted private network.",
				value,
			),
		)
	}
}
