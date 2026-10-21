package validators

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// MsDurationBetween returns a validator that checks a string is a duration in the npm ms format the
// Infisical API reads, such as "7d", "12h" or "2 days", and that it falls between min and max.
func MsDurationBetween(minimum, maximum time.Duration) validator.String {
	return msDurationValidator{minimum: minimum, maximum: maximum}
}

type msDurationValidator struct {
	minimum, maximum time.Duration
}

// No unit means milliseconds.
var msDurationPattern = regexp.MustCompile(`(?i)^(-?(?:\d+)?\.?\d+) *(milliseconds?|msecs?|ms|seconds?|secs?|s|minutes?|mins?|m|hours?|hrs?|h|days?|d|weeks?|w|years?|yrs?|y)?$`)

func parseMsDuration(value string) (time.Duration, bool) {
	if len(value) > 100 {
		return 0, false
	}
	match := msDurationPattern.FindStringSubmatch(value)
	if match == nil {
		return 0, false
	}

	amount, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return 0, false
	}

	unit := time.Millisecond
	switch strings.ToLower(match[2]) {
	case "seconds", "second", "secs", "sec", "s":
		unit = time.Second
	case "minutes", "minute", "mins", "min", "m":
		unit = time.Minute
	case "hours", "hour", "hrs", "hr", "h":
		unit = time.Hour
	case "days", "day", "d":
		unit = 24 * time.Hour
	case "weeks", "week", "w":
		unit = 7 * 24 * time.Hour
	case "years", "year", "yrs", "yr", "y":
		unit = time.Duration(365.25 * float64(24*time.Hour))
	}

	return time.Duration(amount * float64(unit)), true
}

func describeDuration(d time.Duration) string {
	plural := func(n int64, unit string) string {
		if n == 1 {
			return fmt.Sprintf("1 %s", unit)
		}
		return fmt.Sprintf("%d %ss", n, unit)
	}
	switch {
	case d%(24*time.Hour) == 0:
		return plural(int64(d/(24*time.Hour)), "day")
	case d%time.Hour == 0:
		return plural(int64(d/time.Hour), "hour")
	default:
		return plural(int64(d/time.Minute), "minute")
	}
}

func (v msDurationValidator) Description(_ context.Context) string {
	return fmt.Sprintf("must be a duration such as 7d or 12h, between %s and %s", describeDuration(v.minimum), describeDuration(v.maximum))
}

func (v msDurationValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v msDurationValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	duration, ok := parseMsDuration(req.ConfigValue.ValueString())
	if !ok || duration < v.minimum || duration > v.maximum {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid "+req.Path.String(),
			fmt.Sprintf("%s %s, got %q.", req.Path.String(), v.Description(ctx), req.ConfigValue.ValueString()),
		)
	}
}
