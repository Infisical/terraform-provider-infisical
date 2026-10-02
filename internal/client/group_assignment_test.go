package infisicalclient

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
)

// groupAssignmentServer stands in for the organization group-membership endpoints. Retries are
// deliberately not configured, so an error response surfaces immediately.
func groupAssignmentServer(t *testing.T, handlers map[string]http.HandlerFunc) Client {
	t.Helper()

	mux := http.NewServeMux()
	for pattern, handler := range handlers {
		mux.HandleFunc(pattern, handler)
	}

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return Client{Config: Config{
		HostURL:               srv.URL,
		HttpClient:            resty.New().SetBaseURL(srv.URL),
		IsMachineIdentityAuth: true,
	}}
}

const availableGroupsBody = `{"groups":[
	{"id":"11111111-1111-1111-1111-111111111111","name":"Platform","slug":"platform"},
	{"id":"22222222-2222-2222-2222-222222222222","name":"Security","slug":"security"}
]}`

// The available-groups list is the validation gate for linking: a group is resolved by ID when
// one is given, by slug otherwise, and anything not listed is reported as ErrNotFound.
func TestGetAvailableGroup(t *testing.T) {
	client := groupAssignmentServer(t, map[string]http.HandlerFunc{
		"GET /api/v1/organization/available-groups": jsonResponse(http.StatusOK, availableGroupsBody),
	})

	cases := []struct {
		name     string
		groupID  string
		slug     string
		wantID   string
		notFound bool
	}{
		{name: "by id", groupID: "22222222-2222-2222-2222-222222222222", wantID: "22222222-2222-2222-2222-222222222222"},
		{name: "by slug", slug: "platform", wantID: "11111111-1111-1111-1111-111111111111"},
		{name: "id takes precedence over slug", groupID: "22222222-2222-2222-2222-222222222222", slug: "platform", wantID: "22222222-2222-2222-2222-222222222222"},
		{name: "unknown id", groupID: "33333333-3333-3333-3333-333333333333", notFound: true},
		{name: "unknown slug", slug: "nope", notFound: true},
		{name: "slug match is case sensitive", slug: "Platform", notFound: true},
		{name: "nothing given", notFound: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			group, err := client.GetAvailableGroup(tc.groupID, tc.slug)
			if tc.notFound {
				if !errors.Is(err, ErrNotFound) {
					t.Fatalf("expected ErrNotFound, got group %+v, err %v", group, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if group.ID != tc.wantID {
				t.Errorf("expected group %s, got %s", tc.wantID, group.ID)
			}
		})
	}
}

// A failing list (e.g. the session is scoped to the root org) cannot prove the group is absent,
// so it must not be reported as ErrNotFound.
func TestGetAvailableGroupListFails(t *testing.T) {
	client := groupAssignmentServer(t, map[string]http.HandlerFunc{
		"GET /api/v1/organization/available-groups": jsonResponse(http.StatusBadRequest, `{"statusCode":400,"message":"Not a sub-organization","reqId":"req-1"}`),
	})

	_, err := client.GetAvailableGroup("11111111-1111-1111-1111-111111111111", "")
	if err == nil {
		t.Fatal("expected an error when the available-groups list fails")
	}
	if errors.Is(err, ErrNotFound) {
		t.Error("a failed lookup must not be reported as ErrNotFound")
	}
	if got := collectAPIErrors(err); len(got) != 1 || got[0].StatusCode != http.StatusBadRequest {
		t.Errorf("expected the 400 to stay classifiable, got %v", got)
	}
}

// The API validates each role against a strict permanent or temporary shape, so a permanent
// role must carry no temporary fields at all, while a temporary one must carry all of them.
func TestCreateOrgGroupMembershipRequestBody(t *testing.T) {
	var body map[string][]map[string]any
	client := groupAssignmentServer(t, map[string]http.HandlerFunc{
		"POST /api/v1/organizations/memberships/groups/{groupId}": func(w http.ResponseWriter, r *http.Request) {
			if got := r.PathValue("groupId"); got != "11111111-1111-1111-1111-111111111111" {
				t.Errorf("unexpected group id in path: %s", got)
			}
			raw, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Fatalf("request body is not valid JSON: %v", err)
			}
			jsonResponse(http.StatusOK, `{"groupMembership":{"id":"m-1","groupId":"11111111-1111-1111-1111-111111111111","group":{"id":"11111111-1111-1111-1111-111111111111","name":"Platform","slug":"platform","orgId":"root-org"},"roles":[],"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"}}`)(w, r)
		},
	})

	start := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	membership, err := client.CreateOrgGroupMembership(CreateOrgGroupMembershipRequest{
		GroupID: "11111111-1111-1111-1111-111111111111",
		Roles: []OrgGroupMembershipRoleRequest{
			{Role: "member"},
			{Role: "admin", IsTemporary: true, TemporaryMode: "relative", TemporaryRange: "1h", TemporaryAccessStartTime: &start},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if membership.ID != "m-1" || membership.Group.OrgID != "root-org" {
		t.Errorf("unexpected membership decoded: %+v", membership)
	}

	roles := body["roles"]
	if len(roles) != 2 {
		t.Fatalf("expected 2 roles in the request, got %d", len(roles))
	}

	permanent := roles[0]
	if len(permanent) != 2 || permanent["role"] != "member" || permanent["isTemporary"] != false {
		t.Errorf("permanent role must carry only role and isTemporary, got %v", permanent)
	}

	temporary := roles[1]
	for _, key := range []string{"role", "isTemporary", "temporaryMode", "temporaryRange", "temporaryAccessStartTime"} {
		if _, ok := temporary[key]; !ok {
			t.Errorf("temporary role is missing %s: %v", key, temporary)
		}
	}
	if temporary["temporaryAccessStartTime"] != "2026-01-02T03:04:05Z" {
		t.Errorf("unexpected start time: %v", temporary["temporaryAccessStartTime"])
	}
}

func TestGetOrgGroupMembership(t *testing.T) {
	client := groupAssignmentServer(t, map[string]http.HandlerFunc{
		"GET /api/v1/organizations/memberships/groups/linked":  jsonResponse(http.StatusOK, `{"groupMembership":{"id":"m-1","groupId":"linked","group":{"id":"linked","name":"Platform","slug":"platform","orgId":"root-org"},"roles":[{"id":"r-1","role":"custom","customRoleId":"cr-1","customRoleSlug":"deployer","isTemporary":false,"temporaryMode":null,"temporaryRange":null,"temporaryAccessStartTime":null,"temporaryAccessEndTime":null,"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"}],"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z"}}`),
		"GET /api/v1/organizations/memberships/groups/missing": jsonResponse(http.StatusNotFound, `{"statusCode":404,"message":"Group membership not found","reqId":"req-2"}`),
		"GET /api/v1/organizations/memberships/groups/broken":  jsonResponse(http.StatusForbidden, `{"statusCode":403,"message":"Forbidden","reqId":"req-3"}`),
	})

	membership, err := client.GetOrgGroupMembership("linked")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(membership.Roles) != 1 || membership.Roles[0].CustomRoleSlug == nil || *membership.Roles[0].CustomRoleSlug != "deployer" {
		t.Errorf("expected the custom role slug to be decoded, got %+v", membership.Roles)
	}
	if membership.Roles[0].TemporaryAccessStartTime != nil {
		t.Error("a null start time must decode to nil")
	}

	if _, err := client.GetOrgGroupMembership("missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("a 404 must map to ErrNotFound, got %v", err)
	}

	_, err = client.GetOrgGroupMembership("broken")
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Errorf("a 403 must surface as an error other than ErrNotFound, got %v", err)
	}

	if _, err := client.GetOrgGroupMembership(""); !errors.Is(err, ErrNotFound) {
		t.Errorf("an empty group id must map to ErrNotFound, got %v", err)
	}
}

// Slug lookup walks every page of the membership list.
func TestGetOrgGroupMembershipBySlugPaginates(t *testing.T) {
	var offsets []string
	client := groupAssignmentServer(t, map[string]http.HandlerFunc{
		"GET /api/v1/organizations/memberships/groups": func(w http.ResponseWriter, r *http.Request) {
			offset := r.URL.Query().Get("offset")
			offsets = append(offsets, offset)

			memberships := make([]OrgGroupMembership, 0, 100)
			if offset == "0" {
				for i := 0; i < 100; i++ {
					memberships = append(memberships, OrgGroupMembership{ID: "m", GroupID: "g", Group: OrgGroupMembershipGroup{Slug: "filler"}})
				}
			} else {
				memberships = append(memberships, OrgGroupMembership{ID: "m-last", GroupID: "g-last", Group: OrgGroupMembershipGroup{Slug: "target"}})
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(ListOrgGroupMembershipsResponse{GroupMemberships: memberships, TotalCount: 101})
		},
	})

	membership, err := client.GetOrgGroupMembershipBySlug("target")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if membership.GroupID != "g-last" {
		t.Errorf("expected the membership from the second page, got %+v", membership)
	}
	if len(offsets) != 2 || offsets[0] != "0" || offsets[1] != "100" {
		t.Errorf("expected two pages at offsets 0 and 100, got %v", offsets)
	}

	if _, err := client.GetOrgGroupMembershipBySlug("absent"); !errors.Is(err, ErrNotFound) {
		t.Errorf("an unknown slug must map to ErrNotFound, got %v", err)
	}
}
