package validators

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestMsDurationBetween(t *testing.T) {
	v := MsDurationBetween(time.Hour, 30*24*time.Hour)

	cases := map[string]bool{
		"1h":       true,
		"7d":       true,
		"48h":      true,
		"30d":      true,
		"2 days":   true,
		"1.5 Days": true,
		"4w":       true,
		"3600000":  true,
		"59m":      false,
		"31d":      false,
		"5w":       false,
		"1y":       false,
		"abc":      false,
		"7 dayz":   false,
		"":         false,
	}

	for value, wantValid := range cases {
		resp := &validator.StringResponse{}
		v.ValidateString(context.Background(), validator.StringRequest{
			Path:        path.Root("max_request_ttl"),
			ConfigValue: types.StringValue(value),
		}, resp)
		if resp.Diagnostics.HasError() == wantValid {
			t.Errorf("%q: valid = %v, want %v", value, !resp.Diagnostics.HasError(), wantValid)
		}
	}

	for _, c := range []struct {
		minimum, maximum time.Duration
		want             string
	}{
		{time.Hour, 30 * 24 * time.Hour, "must be a duration such as 7d or 12h, between 1 hour and 30 days"},
		{30 * time.Minute, 90 * time.Minute, "must be a duration such as 7d or 12h, between 30 minutes and 90 minutes"},
	} {
		if got := MsDurationBetween(c.minimum, c.maximum).Description(context.Background()); got != c.want {
			t.Errorf("Description() = %q, want %q", got, c.want)
		}
	}
}
