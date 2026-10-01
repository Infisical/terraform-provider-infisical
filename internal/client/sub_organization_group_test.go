package infisicalclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
)

func subOrgGroupServer(t *testing.T, mux *http.ServeMux) Client {
	t.Helper()

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return Client{Config: Config{
		HostURL:               srv.URL,
		HttpClient:            resty.New().SetBaseURL(srv.URL),
		IsMachineIdentityAuth: true,
	}}
}

// Terraform relies on ErrNotFound to know the link is gone.
func TestOrgGroupMembershipNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/organizations/memberships/groups/missing",
		jsonResponse(http.StatusNotFound, `{"statusCode":404,"message":"Group with ID 'missing' not found"}`))
	client := subOrgGroupServer(t, mux)

	if _, err := client.GetOrgGroupMembership("missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound from get, got: %v", err)
	}

	if err := client.DeleteOrgGroupMembership("missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound from delete, got: %v", err)
	}
}

// A 403 must not look like a missing link, or Terraform would drop it from state.
func TestGetOrgGroupMembershipForbidden(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/organizations/memberships/groups/g1",
		jsonResponse(http.StatusForbidden, `{"statusCode":403,"message":"forbidden"}`))
	client := subOrgGroupServer(t, mux)

	_, err := client.GetOrgGroupMembership("g1")
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Errorf("expected a non-ErrNotFound error, got: %v", err)
	}
}

func TestGetOrgGroupMembershipBySlugPaginates(t *testing.T) {
	const total = 150
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/organizations/memberships/groups", func(w http.ResponseWriter, r *http.Request) {
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

		var page []OrgGroupMembership
		for i := offset; i < total && i < offset+100; i++ {
			page = append(page, OrgGroupMembership{
				ID:      fmt.Sprintf("m%d", i),
				GroupID: fmt.Sprintf("g%d", i),
				Group:   OrgGroupMembershipGroup{ID: fmt.Sprintf("g%d", i), Slug: fmt.Sprintf("slug-%d", i)},
			})
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ListOrgGroupMembershipsResponse{GroupMemberships: page, TotalCount: total})
	})
	client := subOrgGroupServer(t, mux)

	membership, err := client.GetOrgGroupMembershipBySlug("slug-120")
	if err != nil {
		t.Fatalf("expected the slug on the second page to be found, got: %v", err)
	}
	if membership.GroupID != "g120" {
		t.Errorf("expected group g120, got %s", membership.GroupID)
	}

	if _, err := client.GetOrgGroupMembershipBySlug("slug-999"); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound for an unknown slug, got: %v", err)
	}
}

func TestGetAvailableGroupBySlug(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/organization/available-groups",
		jsonResponse(http.StatusOK, `{"groups":[{"id":"g1","name":"Platform","slug":"platform"}]}`))
	client := subOrgGroupServer(t, mux)

	group, err := client.GetAvailableGroupBySlug("platform")
	if err != nil || group.ID != "g1" {
		t.Errorf("expected group g1, got %+v (err: %v)", group, err)
	}

	if _, err := client.GetAvailableGroupBySlug("other"); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound for an unknown slug, got: %v", err)
	}
}

// The API rejects temporary fields on a permanent role.
func TestCreateOrgGroupMembershipRequestBody(t *testing.T) {
	// Raw fields so the test can tell an omitted key from an empty one.
	var body struct {
		GroupID *string                      `json:"groupId"`
		Roles   []map[string]json.RawMessage `json:"roles"`
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/organizations/memberships/groups/g1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		jsonResponse(http.StatusOK, `{"groupMembership":{"id":"m1","groupId":"g1","group":{"id":"g1","name":"Platform","slug":"platform"},"roles":[]}}`)(w, r)
	})
	client := subOrgGroupServer(t, mux)

	start := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	membership, err := client.CreateOrgGroupMembership(CreateOrgGroupMembershipRequest{
		GroupID: "g1",
		Roles: []OrgGroupMembershipRoleRequest{
			{Role: "member"},
			{Role: "admin", IsTemporary: true, TemporaryMode: "relative", TemporaryRange: "1h", TemporaryAccessStartTime: &start},
		},
	})
	if err != nil {
		t.Fatalf("expected the link to be created, got: %v", err)
	}
	if membership.ID != "m1" {
		t.Errorf("expected membership m1, got %s", membership.ID)
	}

	if len(body.Roles) != 2 {
		t.Fatalf("expected 2 roles in the body, got %d", len(body.Roles))
	}

	for _, field := range []string{"temporaryMode", "temporaryRange", "temporaryAccessStartTime"} {
		if _, ok := body.Roles[0][field]; ok {
			t.Errorf("permanent role must not send %s", field)
		}
	}
	if body.GroupID != nil {
		t.Error("groupId belongs in the path, not the body")
	}

	if got := string(body.Roles[1]["temporaryAccessStartTime"]); got != `"2026-10-01T09:00:00Z"` {
		t.Errorf("unexpected temporaryAccessStartTime %s", got)
	}
}
