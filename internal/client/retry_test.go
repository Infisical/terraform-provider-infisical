package infisicalclient

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
)

func TestParseRetryAfter(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
		ok   bool
	}{
		{"", 0, false},
		{"  ", 0, false},
		{"0", 0, true},
		{"58", 58 * time.Second, true},
		{"-3", 0, true},
		{"not a number", 0, false},
	}
	for _, c := range cases {
		got, ok := parseRetryAfter(c.in)
		if ok != c.ok || got != c.want {
			t.Errorf("parseRetryAfter(%q) = (%v, %v); want (%v, %v)", c.in, got, ok, c.want, c.ok)
		}
	}

	future := time.Now().Add(45 * time.Second).UTC().Format(http.TimeFormat)
	got, ok := parseRetryAfter(future)
	if !ok || got <= 0 || got > 46*time.Second {
		t.Errorf("parseRetryAfter(HTTP-date future) = (%v, %v); want positive <= 46s", got, ok)
	}

	past := time.Now().Add(-45 * time.Second).UTC().Format(http.TimeFormat)
	got, ok = parseRetryAfter(past)
	if !ok || got != 0 {
		t.Errorf("parseRetryAfter(HTTP-date past) = (%v, %v); want (0, true)", got, ok)
	}
}

func TestParseRateLimitReset(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
		ok   bool
	}{
		{"", 0, false},
		{"abc", 0, false},
		{"30", 30 * time.Second, true},
		{"-1", 0, true},
	}
	for _, c := range cases {
		got, ok := parseRateLimitReset(c.in)
		if ok != c.ok || got != c.want {
			t.Errorf("parseRateLimitReset(%q) = (%v, %v); want (%v, %v)", c.in, got, ok, c.want, c.ok)
		}
	}

	absolute := fmt.Sprintf("%d", time.Now().Add(20*time.Second).Unix())
	got, ok := parseRateLimitReset(absolute)
	if !ok || got <= 0 || got > 21*time.Second {
		t.Errorf("parseRateLimitReset(absolute future) = (%v, %v); want positive <= 21s", got, ok)
	}
}

func TestConfigureRetries_Retries429AndHonorsRetryAfter(t *testing.T) {
	var count int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&count, 1)
		if n < 3 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"message":"rate limited"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	rc := resty.New()
	rc.SetBaseURL(server.URL)
	configureRetries(rc)

	start := time.Now()
	resp, err := rc.R().Get("/")
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode(), resp.String())
	}
	if got := atomic.LoadInt32(&count); got != 3 {
		t.Fatalf("handler calls = %d, want 3", got)
	}
	if elapsed := time.Since(start); elapsed < 2*time.Second {
		t.Fatalf("elapsed = %v, want at least 2s (Retry-After honored)", elapsed)
	}
}

func TestConfigureRetries_DoesNotRetryOn4xxOtherThan429(t *testing.T) {
	var count int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&count, 1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	rc := resty.New()
	rc.SetBaseURL(server.URL)
	configureRetries(rc)

	_, err := rc.R().Get("/")
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	if got := atomic.LoadInt32(&count); got != 1 {
		t.Fatalf("handler calls = %d, want 1 (no retries on 400)", got)
	}
}
