package datasource

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/go-resty/resty/v2"
)

// identitySearchClient returns a client whose identity search endpoint always answers
// with body.
func identitySearchClient(t *testing.T, body string) *infisical.Client {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/identities/search", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return &infisical.Client{Config: infisical.Config{
		HostURL:               srv.URL,
		HttpClient:            resty.New().SetBaseURL(srv.URL),
		IsMachineIdentityAuth: true,
	}}
}

func TestResolveIdentityIDByNameReturnsTheSoleMatch(t *testing.T) {
	client := identitySearchClient(t,
		`{"identities":[{"identity":{"id":"11111111-1111-1111-1111-111111111111","name":"humanitec"}}],"totalCount":1}`)

	id, diags := resolveIdentityIDByName(client, "humanitec")

	if diags.HasError() {
		t.Fatalf("an unambiguous name must resolve, got %v", diags.Errors())
	}
	if id != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("expected the matched identity's id, got %q", id)
	}
}

func TestResolveIdentityIDByNameRejectsNoMatch(t *testing.T) {
	client := identitySearchClient(t, `{"identities":[],"totalCount":0}`)

	id, diags := resolveIdentityIDByName(client, "absent")

	if !diags.HasError() {
		t.Fatal("a name that matches nothing must be an error, not an empty result")
	}
	if id != "" {
		t.Errorf("no id may be returned alongside an error, got %q", id)
	}
	if !strings.Contains(diags.Errors()[0].Detail(), "absent") {
		t.Errorf("the error should name what was looked for, got %q", diags.Errors()[0].Detail())
	}
}

// This is the case the whole design turns on. Identity names are not unique, so a
// name can match several identities; resolving it to one of them anyway would make
// the same configuration bind to a different identity as unrelated identities come
// and go. The error has to be actionable too: the caller's way out is to switch to
// id, so every candidate id belongs in the message.
func TestResolveIdentityIDByNameRefusesToPickAmongDuplicates(t *testing.T) {
	client := identitySearchClient(t, `{"identities":[
		{"identity":{"id":"11111111-1111-1111-1111-111111111111","name":"deploy"}},
		{"identity":{"id":"22222222-2222-2222-2222-222222222222","name":"deploy"}}
	],"totalCount":2}`)

	id, diags := resolveIdentityIDByName(client, "deploy")

	if !diags.HasError() {
		t.Fatal("an ambiguous name must be an error: silently choosing a match makes the lookup non-deterministic")
	}
	if id != "" {
		t.Errorf("no id may be returned for an ambiguous name, got %q", id)
	}

	detail := diags.Errors()[0].Detail()
	for _, want := range []string{
		"11111111-1111-1111-1111-111111111111",
		"22222222-2222-2222-2222-222222222222",
	} {
		if !strings.Contains(detail, want) {
			t.Errorf("the error must list candidate %s so the user can switch to id, got %q", want, detail)
		}
	}
}

// The search reads a single page, so a name shared by more identities than fit on it
// yields a partial list. Presenting that list as the full set would understate the
// problem, so the remainder has to be acknowledged.
func TestResolveIdentityIDByNameReportsDuplicatesBeyondThePage(t *testing.T) {
	client := identitySearchClient(t, `{"identities":[
		{"identity":{"id":"11111111-1111-1111-1111-111111111111","name":"deploy"}},
		{"identity":{"id":"22222222-2222-2222-2222-222222222222","name":"deploy"}}
	],"totalCount":9}`)

	_, diags := resolveIdentityIDByName(client, "deploy")

	if !diags.HasError() {
		t.Fatal("expected an error for an ambiguous name")
	}

	detail := diags.Errors()[0].Detail()
	if !strings.Contains(detail, "9") {
		t.Errorf("the error should report the true number of matches, got %q", detail)
	}
	if !strings.Contains(detail, "7 more") {
		t.Errorf("the error should say the listed ids are only part of the matches, got %q", detail)
	}
}

// A search the API rejects says nothing about whether the name exists. Reporting it
// as "not found" would send someone hunting for a missing identity over a permissions
// or connectivity problem.
func TestResolveIdentityIDByNameDistinguishesFailureFromAbsence(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/identities/search", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"statusCode":403,"message":"You do not have permission to read identities"}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	client := &infisical.Client{Config: infisical.Config{
		HostURL:               srv.URL,
		HttpClient:            resty.New().SetBaseURL(srv.URL),
		IsMachineIdentityAuth: true,
	}}

	_, diags := resolveIdentityIDByName(client, "deploy")

	if !diags.HasError() {
		t.Fatal("expected an error when the search is rejected")
	}
	if summary := diags.Errors()[0].Summary(); strings.Contains(strings.ToLower(summary), "not found") {
		t.Errorf("a rejected search must not be reported as a missing identity, got summary %q", summary)
	}
}
