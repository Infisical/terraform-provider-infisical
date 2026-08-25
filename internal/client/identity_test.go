package infisicalclient

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	infisicalerrors "terraform-provider-infisical/internal/errors"

	"github.com/go-resty/resty/v2"
)

// identitySearchServer stands in for the identity search endpoint and hands back the
// request body the client sent. The outgoing filter is otherwise invisible -- the
// client returns only the decoded response -- and the filter is the part of this call
// that determines whether a name lookup is correct.
func identitySearchServer(t *testing.T, status int, body string) (Client, *[]byte) {
	t.Helper()

	var captured []byte

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/identities/search", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("the search endpoint is a POST, got %s", r.Method)
		}

		sent, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading request body: %v", err)
			return
		}
		captured = sent

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return Client{Config: Config{
		HostURL:               srv.URL,
		HttpClient:            resty.New().SetBaseURL(srv.URL),
		IsMachineIdentityAuth: true,
	}}, &captured
}

// A lookup by name has to mean that exact name. The API also accepts $contains and
// $in, either of which would let "prod" resolve to an identity named "prod-legacy":
// the operator is the difference between a lookup that is merely ambiguous and one
// that is quietly wrong. The page size matters for the same reason -- a page that
// cannot hold a second match makes duplicate names look unique to the caller.
func TestSearchIdentitiesByNameSendsExactMatchFilter(t *testing.T) {
	client, captured := identitySearchServer(t, http.StatusOK, `{"identities":[],"totalCount":0}`)

	if _, err := client.SearchIdentitiesByName("prod"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var sent struct {
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
		Search struct {
			Name map[string]any `json:"name"`
		} `json:"search"`
	}
	if err := json.Unmarshal(*captured, &sent); err != nil {
		t.Fatalf("request body is not the JSON the API expects: %v (%s)", err, *captured)
	}

	if got, ok := sent.Search.Name["$eq"]; !ok || got != "prod" {
		t.Errorf("search.name.$eq must carry the requested name, got %v", sent.Search.Name)
	}
	for _, loose := range []string{"$contains", "$in"} {
		if _, ok := sent.Search.Name[loose]; ok {
			t.Errorf("%s matches names other than the one asked for, so it must not be sent", loose)
		}
	}
	if sent.Limit < 2 {
		t.Errorf("a page must be able to hold more than one match or duplicates look unique, got limit %d", sent.Limit)
	}
}

// Only the first page is fetched, so len(Identities) understates the duplicates once
// they overflow one page. TotalCount is the number a caller has to report, and the
// nested identity has to decode, since the ID inside it is the whole point of the call.
func TestSearchIdentitiesByNameKeepsTotalCountDistinctFromPage(t *testing.T) {
	client, _ := identitySearchServer(t, http.StatusOK,
		`{"identities":[{"identity":{"id":"id-1","name":"dup"}},{"identity":{"id":"id-2","name":"dup"}}],"totalCount":7}`)

	result, err := client.SearchIdentitiesByName("dup")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Identities) != 2 {
		t.Errorf("expected the 2 identities on the page, got %d", len(result.Identities))
	}
	if result.TotalCount != 7 {
		t.Errorf("totalCount must survive decoding so callers can report the real number of duplicates, got %d", result.TotalCount)
	}
	if result.Identities[0].Identity.ID != "id-1" {
		t.Errorf("the nested identity id must decode, got %q", result.Identities[0].Identity.ID)
	}
}

// A search that fails has not established that the name is unused. Reporting it as
// ErrNotFound would let a caller turn a permissions problem into "no such identity".
func TestSearchIdentitiesByNameDoesNotDisguiseFailureAsAbsence(t *testing.T) {
	client, _ := identitySearchServer(t, http.StatusForbidden,
		`{"statusCode":403,"message":"You do not have permission to read identities","reqId":"req-search"}`)

	_, err := client.SearchIdentitiesByName("prod")
	if err == nil {
		t.Fatal("expected an error when the search is rejected")
	}

	if errors.Is(err, ErrNotFound) {
		t.Error("a rejected search must not be reported as ErrNotFound: it cannot prove absence")
	}

	var apiErr *infisicalerrors.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("the API failure must stay classifiable, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusForbidden {
		t.Errorf("expected the 403 to be preserved, got %d", apiErr.StatusCode)
	}
}
