package infisicalclient

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	infisicalerrors "terraform-provider-infisical/internal/errors"

	"github.com/go-resty/resty/v2"
)

// orgLookupServer stands in for the two endpoints GetOrganizationBySlug consults.
// A handler is registered per path so a test can fail one source, both, or neither.
// Retries are deliberately not configured: these tests assert error plumbing, and a
// 5xx would otherwise be retried before the error surfaces.
func orgLookupServer(t *testing.T, details, subOrgs http.HandlerFunc) Client {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/identities/details", details)
	mux.HandleFunc("/api/v1/sub-organizations", subOrgs)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return Client{Config: Config{
		HostURL:               srv.URL,
		HttpClient:            resty.New().SetBaseURL(srv.URL),
		IsMachineIdentityAuth: true,
	}}
}

// collectAPIErrors gathers every APIError in an error tree, walking it the way
// errors.Is and errors.As do: the error itself, then each child, pre-order and
// depth-first, through both wrapping forms the errors package recognises
// (Unwrap() error and Unwrap() []error). errors.As alone cannot do this -- it stops at
// the first match, and both causes here are the same concrete type -- while asserting
// on the top-level error's own shape would pin an implementation detail rather than the
// property that matters: both causes stay reachable. Note errors.Unwrap() is not usable
// for the walk either, since it only calls "Unwrap() error" and so does not descend
// into joined errors.
func collectAPIErrors(err error) []*infisicalerrors.APIError {
	var found []*infisicalerrors.APIError

	var walk func(error)
	walk = func(e error) {
		if e == nil {
			return
		}
		if apiErr, ok := e.(*infisicalerrors.APIError); ok {
			found = append(found, apiErr)
		}
		switch x := e.(type) {
		case interface{ Unwrap() error }:
			walk(x.Unwrap())
		case interface{ Unwrap() []error }:
			for _, inner := range x.Unwrap() {
				walk(inner)
			}
		}
	}
	walk(err)

	return found
}

func jsonResponse(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		fmt.Fprint(w, body)
	}
}

// A machine identity created inside a sub-organization is rejected by both sources:
// the details endpoint 500s because the identity belongs to no root organization, and
// the sub-organization list is authorized against that same root. Neither cause may be
// flattened into text -- a caller must still be able to classify both -- and the result
// must not masquerade as ErrNotFound, since an unavailable source cannot prove absence.
func TestGetOrganizationBySlugBothLookupsFail(t *testing.T) {
	client := orgLookupServer(t,
		jsonResponse(http.StatusInternalServerError, `{"statusCode":500,"message":"Something went wrong","reqId":"req-details"}`),
		jsonResponse(http.StatusForbidden, `{"statusCode":403,"message":"You are not a member of this organization","reqId":"req-list"}`),
	)

	_, err := client.GetOrganizationBySlug("some-sub-org")
	if err == nil {
		t.Fatal("expected an error when both organization lookups fail")
	}

	if errors.Is(err, ErrNotFound) {
		t.Error("a failed lookup must not be reported as ErrNotFound: an unavailable source cannot prove absence")
	}

	// Both causes must stay classifiable, so a caller can tell a 403 from a 500.
	// What matters is that both are reachable in the error tree, not how the tree is
	// shaped: two %w verbs, a wrapper around errors.Join, or any other arrangement
	// errors.Is/errors.As can traverse are all acceptable.
	statuses := map[int]bool{}
	for _, apiErr := range collectAPIErrors(err) {
		statuses[apiErr.StatusCode] = true
	}
	for _, want := range []int{http.StatusInternalServerError, http.StatusForbidden} {
		if !statuses[want] {
			t.Errorf("status %d is not discoverable in the error tree of %T: %v", want, err, err)
		}
	}

	// errors.As on the joined error must reach a cause too, not just the leaves.
	var apiErr *infisicalerrors.APIError
	if !errors.As(err, &apiErr) {
		t.Error("errors.As must traverse the joined error to reach an APIError cause")
	}

	// The actionable guidance stays in the message the user actually reads.
	if got := err.Error(); !strings.Contains(got, "created inside a sub-organization") {
		t.Errorf("error message should explain the sub-organization identity cause, got: %s", got)
	}
}

// Guards the guard: the dual-failure assertion must hold for any tree errors.Is/As can
// traverse, so that preserving both causes differently -- e.g. a contextual wrapper
// around errors.Join instead of two %w verbs -- is not a test failure.
func TestCollectAPIErrorsWalksBothWrappingForms(t *testing.T) {
	first := &infisicalerrors.APIError{Operation: "CallGetIdentityDetails", StatusCode: http.StatusInternalServerError}
	second := &infisicalerrors.APIError{Operation: "CallListSubOrganizations", StatusCode: http.StatusForbidden}

	shapes := map[string]error{
		"two %w verbs":              fmt.Errorf("context: %w and %w", first, second),
		"wrapper around Join":       fmt.Errorf("context: %w", errors.Join(first, second)),
		"nested single wraps":       fmt.Errorf("outer: %w", fmt.Errorf("inner: %w and %w", first, second)),
		"join of wrapped errors":    errors.Join(fmt.Errorf("a: %w", first), fmt.Errorf("b: %w", second)),
		"single cause, single wrap": fmt.Errorf("context: %w", first),
	}

	wantCount := map[string]int{
		"two %w verbs":              2,
		"wrapper around Join":       2,
		"nested single wraps":       2,
		"join of wrapped errors":    2,
		"single cause, single wrap": 1,
	}

	for name, err := range shapes {
		t.Run(name, func(t *testing.T) {
			got := collectAPIErrors(err)
			if len(got) != wantCount[name] {
				t.Fatalf("expected %d APIError(s) in the tree, got %d", wantCount[name], len(got))
			}
			for _, apiErr := range got {
				if apiErr.StatusCode == 0 {
					t.Error("collected an APIError with no status code")
				}
			}
		})
	}

	if collectAPIErrors(nil) != nil {
		t.Error("a nil error has no causes to collect")
	}
	if got := collectAPIErrors(errors.New("plain")); got != nil {
		t.Errorf("an unwrapped non-API error yields no causes, got %v", got)
	}
}

// Only the details endpoint fails: the slug is genuinely absent from the sub-org list,
// but absence cannot be claimed while a source is down, so the cause must surface.
func TestGetOrganizationBySlugDetailsFailureIsNotAbsence(t *testing.T) {
	client := orgLookupServer(t,
		jsonResponse(http.StatusInternalServerError, `{"statusCode":500,"message":"boom","reqId":"req-details"}`),
		jsonResponse(http.StatusOK, `{"organizations":[],"totalCount":0}`),
	)

	_, err := client.GetOrganizationBySlug("missing")
	if err == nil {
		t.Fatal("expected an error when the identity organization lookup fails")
	}
	if errors.Is(err, ErrNotFound) {
		t.Error("must not report ErrNotFound while the identity organization lookup is failing")
	}

	var apiErr *infisicalerrors.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("cause should stay classifiable, got %T", err)
	}
	if apiErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected the 500 cause to surface, got %d", apiErr.StatusCode)
	}
}

// Only the sub-organization list fails, and the slug does not match the identity's own
// organization: same rule, the cause must surface rather than a bare not-found.
func TestGetOrganizationBySlugListFailureIsNotAbsence(t *testing.T) {
	client := orgLookupServer(t,
		jsonResponse(http.StatusOK, `{"identityDetails":{"organization":{"id":"11111111-1111-1111-1111-111111111111","name":"Root","slug":"root-org"}}}`),
		jsonResponse(http.StatusForbidden, `{"statusCode":403,"message":"forbidden","reqId":"req-list"}`),
	)

	_, err := client.GetOrganizationBySlug("some-sub-org")
	if err == nil {
		t.Fatal("expected an error when the sub-organization lookup fails")
	}
	if errors.Is(err, ErrNotFound) {
		t.Error("must not report ErrNotFound while the sub-organization lookup is failing")
	}

	var apiErr *infisicalerrors.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("cause should stay classifiable, got %T", err)
	}
	if apiErr.StatusCode != http.StatusForbidden {
		t.Errorf("expected the 403 cause to surface, got %d", apiErr.StatusCode)
	}
}

// ErrNotFound is reserved for the one case that proves absence: both sources answered
// and neither carried the slug. The data source relies on this to raise "not found"
// rather than an opaque failure.
func TestGetOrganizationBySlugNotFoundOnlyWhenBothSourcesAnswer(t *testing.T) {
	client := orgLookupServer(t,
		jsonResponse(http.StatusOK, `{"identityDetails":{"organization":{"id":"11111111-1111-1111-1111-111111111111","name":"Root","slug":"root-org"}}}`),
		jsonResponse(http.StatusOK, `{"organizations":[{"id":"22222222-2222-2222-2222-222222222222","name":"Sub","slug":"sub-org","parentOrgId":"11111111-1111-1111-1111-111111111111"}],"totalCount":1}`),
	)

	if _, err := client.GetOrganizationBySlug("nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound when both lookups succeed and neither matches, got %v", err)
	}
}

// The two resolution paths: the identity's own organization and the sub-organization
// list. Matching is exact -- a case variant must not resolve.
func TestGetOrganizationBySlugResolves(t *testing.T) {
	details := jsonResponse(http.StatusOK, `{"identityDetails":{"organization":{"id":"11111111-1111-1111-1111-111111111111","name":"Root","slug":"root-org"}}}`)
	subOrgs := jsonResponse(http.StatusOK, `{"organizations":[{"id":"22222222-2222-2222-2222-222222222222","name":"Sub","slug":"sub-org","parentOrgId":"11111111-1111-1111-1111-111111111111"}],"totalCount":1}`)

	cases := []struct {
		name     string
		slug     string
		wantID   string
		wantName string
		wantErr  bool
	}{
		{name: "own organization", slug: "root-org", wantID: "11111111-1111-1111-1111-111111111111", wantName: "Root"},
		{name: "sub-organization", slug: "sub-org", wantID: "22222222-2222-2222-2222-222222222222", wantName: "Sub"},
		{name: "case variant does not match", slug: "Sub-Org", wantErr: true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			client := orgLookupServer(t, details, subOrgs)
			org, err := client.GetOrganizationBySlug(c.slug)
			if c.wantErr {
				if !errors.Is(err, ErrNotFound) {
					t.Fatalf("expected ErrNotFound for %q, got %v", c.slug, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if org.ID != c.wantID || org.Name != c.wantName || org.Slug != c.slug {
				t.Errorf("got %+v, want id=%s name=%s slug=%s", org, c.wantID, c.wantName, c.slug)
			}
		})
	}
}
