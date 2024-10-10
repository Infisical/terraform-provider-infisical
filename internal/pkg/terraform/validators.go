package terraform

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
)

var SlugRegexValidator = stringvalidator.RegexMatches(
	regexp.MustCompile(`^[a-z0-9_-]*$`),
	"invalid slug, slugs must be lowercase alphanumeric characters and hyphens only (example-slug-1)",
)
