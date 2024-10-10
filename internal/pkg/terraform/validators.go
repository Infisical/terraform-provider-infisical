package terraform

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
)

var LowercaseRegexValidator = stringvalidator.RegexMatches(
	regexp.MustCompile(`^(([^a-zA-Z0-9])|([a-z0-9]))*$`),
	"alphanumeric characters must be lowercase",
)
