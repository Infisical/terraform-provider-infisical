package infisicalclient

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	infisicalerrors "terraform-provider-infisical/internal/errors"
)

// Helpers shared by the client's tests. They live here rather than in whichever test file first
// needed them, so a second consumer does not have to reach into an unrelated feature's file.

func jsonResponse(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		fmt.Fprint(w, body)
	}
}

// collectAPIErrors gathers every APIError in an error tree, walking it the way
// errors.Is and errors.As do: the error itself, then each child, pre-order and
// depth-first, through both wrapping forms the errors package recognises
// (Unwrap() error and Unwrap() []error). errors.As alone cannot do this -- it stops at
// the first match, and several causes may share the one concrete type -- while asserting
// on the top-level error's own shape would pin an implementation detail rather than the
// property that matters: every cause stays reachable. Note errors.Unwrap() is not usable
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

// Guards the guard: assertions built on collectAPIErrors must hold for any tree errors.Is/As can
// traverse, so that preserving causes differently -- e.g. a contextual wrapper around errors.Join
// instead of two %w verbs -- is not a test failure.
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
