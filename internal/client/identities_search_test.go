package infisicalclient

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
)

func newIdentitySearchTestClient(t *testing.T, handler http.HandlerFunc) (Client, *httptest.Server) {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	httpClient := resty.New()
	httpClient.SetBaseURL(server.URL)

	return Client{Config: Config{HttpClient: httpClient}}, server
}

func identitySearchMatch(id, identityID, scope string, projectID *string) IdentitySearchMatch {
	match := IdentitySearchMatch{
		ID:         id,
		IdentityID: identityID,
		Scope:      scope,
		ProjectID:  projectID,
		Identity: IdentitySearchIdentity{
			ID:   identityID,
			Name: "test-identity",
		},
	}
	return match
}

func writeIdentitySearchResponse(t *testing.T, w http.ResponseWriter, identities []IdentitySearchMatch, totalCount int) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(identitySearchResponse{
		Identities: identities,
		TotalCount: totalCount,
	}); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func TestSearchIdentitiesByName_PaginatesAcrossMultiplePages(t *testing.T) {
	var offsets []int

	client, _ := newIdentitySearchTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/identities/search" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}

		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}

		offset, ok := payload["offset"].(float64)
		if !ok {
			t.Fatalf("offset missing from request body: %v", payload)
		}
		offsets = append(offsets, int(offset))

		switch int(offset) {
		case 0:
			writeIdentitySearchResponse(t, w, []IdentitySearchMatch{
				identitySearchMatch("match-1", "identity-1", "organization", nil),
				identitySearchMatch("match-2", "identity-2", "organization", nil),
			}, 3)
		case 2:
			writeIdentitySearchResponse(t, w, []IdentitySearchMatch{
				identitySearchMatch("match-3", "identity-3", "project", strPtr("project-1")),
			}, 3)
		default:
			t.Fatalf("unexpected offset: %d", int(offset))
		}
	})

	identities, totalCount, err := client.SearchIdentitiesByName(SearchIdentityIDsByNameRequest{
		IdentityName: "iac",
		Limit:        2,
	})
	if err != nil {
		t.Fatalf("SearchIdentitiesByName() error = %v", err)
	}
	if totalCount != 3 {
		t.Fatalf("totalCount = %d, want 3", totalCount)
	}
	if len(identities) != 3 {
		t.Fatalf("len(identities) = %d, want 3", len(identities))
	}
	if got := []int{offsets[0], offsets[1]}; got[0] != 0 || got[1] != 2 {
		t.Fatalf("requested offsets = %v, want [0 2]", offsets)
	}
}

func TestSearchIdentitiesByName_DeduplicatesDuplicateMatchIDs(t *testing.T) {
	client, _ := newIdentitySearchTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeIdentitySearchResponse(t, w, []IdentitySearchMatch{
			identitySearchMatch("match-1", "identity-1", "organization", nil),
			identitySearchMatch("match-1", "identity-1", "organization", nil),
		}, 2)
	})

	identities, totalCount, err := client.SearchIdentitiesByName(SearchIdentityIDsByNameRequest{
		IdentityName: "iac",
	})
	if err != nil {
		t.Fatalf("SearchIdentitiesByName() error = %v", err)
	}
	if totalCount != 2 {
		t.Fatalf("totalCount = %d, want 2", totalCount)
	}
	if len(identities) != 1 {
		t.Fatalf("len(identities) = %d, want 1", len(identities))
	}
	if identities[0].ID != "match-1" {
		t.Fatalf("identities[0].ID = %q, want match-1", identities[0].ID)
	}
}

func TestSearchIdentitiesByName_StopsOnEmptyIntermediatePage(t *testing.T) {
	var requestCount int

	client, _ := newIdentitySearchTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}

		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}

		offset, ok := payload["offset"].(float64)
		if !ok {
			t.Fatalf("offset missing from request body: %v", payload)
		}

		switch int(offset) {
		case 0:
			writeIdentitySearchResponse(t, w, []IdentitySearchMatch{
				identitySearchMatch("match-1", "identity-1", "organization", nil),
				identitySearchMatch("match-2", "identity-2", "organization", nil),
			}, 5)
		case 2:
			writeIdentitySearchResponse(t, w, nil, 5)
		default:
			t.Fatalf("unexpected offset: %d", int(offset))
		}
	})

	identities, totalCount, err := client.SearchIdentitiesByName(SearchIdentityIDsByNameRequest{
		IdentityName: "iac",
		Limit:        2,
	})
	if err != nil {
		t.Fatalf("SearchIdentitiesByName() error = %v", err)
	}
	if requestCount != 2 {
		t.Fatalf("requestCount = %d, want 2", requestCount)
	}
	if totalCount != 5 {
		t.Fatalf("totalCount = %d, want 5", totalCount)
	}
	if len(identities) != 2 {
		t.Fatalf("len(identities) = %d, want 2", len(identities))
	}
}

func TestSearchIdentitiesByName_FallbackDedupKeyPreservesDistinctMemberships(t *testing.T) {
	projectA := "project-a"
	projectB := "project-b"

	client, _ := newIdentitySearchTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeIdentitySearchResponse(t, w, []IdentitySearchMatch{
			{
				IdentityID: "",
				Scope:      "project",
				ProjectID:  &projectA,
				Identity:   IdentitySearchIdentity{ID: "identity-1", Name: "iac"},
			},
			{
				IdentityID: "",
				Scope:      "project",
				ProjectID:  &projectB,
				Identity:   IdentitySearchIdentity{ID: "identity-1", Name: "iac"},
			},
			{
				IdentityID: "",
				Scope:      "project",
				ProjectID:  &projectA,
				Identity:   IdentitySearchIdentity{ID: "identity-1", Name: "iac"},
			},
		}, 3)
	})

	identities, totalCount, err := client.SearchIdentitiesByName(SearchIdentityIDsByNameRequest{
		IdentityName: "iac",
	})
	if err != nil {
		t.Fatalf("SearchIdentitiesByName() error = %v", err)
	}
	if totalCount != 3 {
		t.Fatalf("totalCount = %d, want 3", totalCount)
	}
	if len(identities) != 2 {
		t.Fatalf("len(identities) = %d, want 2", len(identities))
	}
	if identities[0].IdentityID != "identity-1" || identities[1].IdentityID != "identity-1" {
		t.Fatalf("expected identityId to be backfilled from nested identity.id")
	}
}

func strPtr(value string) *string {
	return &value
}
