package infisicalclient

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	defaultRetryCount       = 5
	defaultRetryWaitTime    = 1 * time.Second
	defaultRetryMaxWaitTime = 60 * time.Second
)

// configureRetries installs retry behavior on the resty client so transient
// errors (notably 429 Too Many Requests) do not fail the Terraform run.
// The Retry-After header is honored when present; otherwise jittered
// exponential backoff is used, capped at defaultRetryMaxWaitTime.
func configureRetries(c *resty.Client) {
	c.SetRetryCount(defaultRetryCount)
	c.SetRetryWaitTime(defaultRetryWaitTime)
	c.SetRetryMaxWaitTime(defaultRetryMaxWaitTime)

	c.AddRetryCondition(func(r *resty.Response, err error) bool {
		if r == nil {
			return false
		}
		switch r.StatusCode() {
		case http.StatusTooManyRequests,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout:
			return true
		}
		return false
	})

	c.SetRetryAfter(func(_ *resty.Client, r *resty.Response) (time.Duration, error) {
		if r == nil {
			return 0, nil
		}
		if d, ok := parseRetryAfter(r.Header().Get("Retry-After")); ok {
			return d, nil
		}
		if d, ok := parseRateLimitReset(r.Header().Get("X-RateLimit-Reset")); ok {
			return d, nil
		}
		return 0, nil
	})
}

// parseRetryAfter parses an HTTP Retry-After header. It supports both the
// delta-seconds and HTTP-date forms defined by RFC 7231.
func parseRetryAfter(v string) (time.Duration, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, false
	}
	if secs, err := strconv.Atoi(v); err == nil {
		if secs < 0 {
			secs = 0
		}
		return time.Duration(secs) * time.Second, true
	}
	if t, err := http.ParseTime(v); err == nil {
		d := time.Until(t)
		if d < 0 {
			d = 0
		}
		return d, true
	}
	return 0, false
}

// parseRateLimitReset parses the X-RateLimit-Reset header, which some APIs
// send instead of (or alongside) Retry-After. The value is either a
// delta-seconds integer or an absolute Unix timestamp; both are supported.
func parseRateLimitReset(v string) (time.Duration, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, false
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, false
	}
	if n < 0 {
		return 0, true
	}
	// Values above ~year 2001 in unix seconds are absolute timestamps;
	// smaller numbers are delta-seconds.
	const absoluteTimestampThreshold = 1_000_000_000
	if n >= absoluteTimestampThreshold {
		d := time.Until(time.Unix(n, 0))
		if d < 0 {
			d = 0
		}
		return d, true
	}
	return time.Duration(n) * time.Second, true
}
