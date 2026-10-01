package terraform

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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

// IDs come back from the API in lowercase, so an uppercase spelling would never match state.
var UuidValidator = stringvalidator.RegexMatches(
	regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`),
	"must be a lowercase UUID (example: 3f1c9a8e-2b4d-4c6e-9f10-1a2b3c4d5e6f). To leave it unset, omit the attribute or use null",
)
