package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const groupUserTestGroupID = "3f1c9a8e-2b4d-4c6e-9f10-1a2b3c4d5e6f"

type fakeGroupUserBackend struct {
	t            *testing.T
	users        map[string]string
	members      map[string]map[string]bool
	forbidDelete bool
	requests     []string
}

func newFakeGroupUserBackend(t *testing.T) *fakeGroupUserBackend {
	return &fakeGroupUserBackend{
		t: t,
		users: map[string]string{
			"alice@example.com":     "u-alice",
			"bob@example.com":       "u-bob",
			"bob@example.community": "u-bob2",
		},
		members: map[string]map[string]bool{
			groupUserTestGroupID: {},
		},
	}
}

func (b *fakeGroupUserBackend) writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		b.t.Error(err)
	}
}

func (b *fakeGroupUserBackend) apiError(w http.ResponseWriter, status int, message string) {
	b.writeJSON(w, status, map[string]any{"statusCode": status, "message": message})
}

func (b *fakeGroupUserBackend) requestsMatching(method string) int {
	count := 0
	for _, request := range b.requests {
		if strings.HasPrefix(request, method+" ") {
			count++
		}
	}
	return count
}

func (b *fakeGroupUserBackend) handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/groups/{id}/users/{username}", func(w http.ResponseWriter, r *http.Request) {
		groupID, username := r.PathValue("id"), r.PathValue("username")
		members, ok := b.members[groupID]
		if !ok {
			b.apiError(w, http.StatusNotFound, "Failed to find group with ID "+groupID)
			return
		}
		userID, ok := b.users[username]
		if !ok {
			b.apiError(w, http.StatusNotFound, "Failed to find user with username "+username)
			return
		}
		if members[username] {
			b.apiError(w, http.StatusBadRequest, "User(s) are already part of the group "+groupID)
			return
		}
		members[username] = true
		b.writeJSON(w, http.StatusOK, map[string]any{"id": userID, "username": username, "email": username})
	})

	mux.HandleFunc("DELETE /api/v1/groups/{id}/users/{username}", func(w http.ResponseWriter, r *http.Request) {
		if b.forbidDelete {
			b.apiError(w, http.StatusForbidden, "You are not allowed to perform this action")
			return
		}
		groupID, username := r.PathValue("id"), r.PathValue("username")
		members, ok := b.members[groupID]
		if !ok {
			b.apiError(w, http.StatusNotFound, "Failed to find group with ID "+groupID)
			return
		}
		userID, ok := b.users[username]
		if !ok {
			b.apiError(w, http.StatusNotFound, "Failed to find user with username "+username)
			return
		}
		if !members[username] {
			b.apiError(w, http.StatusForbidden, "User(s) are not part of the group "+groupID)
			return
		}
		delete(members, username)
		b.writeJSON(w, http.StatusOK, map[string]any{"id": userID, "username": username, "email": username})
	})

	// The tests never exceed one page, so the fake ignores offset and limit. Paging is covered by
	// the client's TestListGroupUsersPaginates.
	mux.HandleFunc("GET /api/v1/groups/{id}/users", func(w http.ResponseWriter, r *http.Request) {
		groupID := r.PathValue("id")
		members, ok := b.members[groupID]
		if !ok {
			b.apiError(w, http.StatusNotFound, "Failed to find group with ID "+groupID)
			return
		}

		search := strings.ToLower(r.URL.Query().Get("username"))
		existingMembersOnly := r.URL.Query().Get("filter") == "existingMembers"

		matches := []infisical.GroupUser{}
		for username, userID := range b.users {
			isMember := members[username]
			if existingMembersOnly && !isMember {
				continue
			}

			if !strings.Contains(strings.ToLower(username), search) {
				continue
			}
			matches = append(matches, infisical.GroupUser{ID: userID, Username: username, IsPartOfGroup: isMember})
		}
		b.writeJSON(w, http.StatusOK, infisical.ListGroupUsersResponse{GroupUsers: matches, TotalCount: len(matches)})
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b.requests = append(b.requests, r.Method+" "+r.URL.Path)
		mux.ServeHTTP(w, r)
	})
}

func newGroupUserTestResource(t *testing.T, backend *fakeGroupUserBackend) *GroupUserResource {
	t.Helper()
	server := httptest.NewServer(backend.handler())
	t.Cleanup(server.Close)
	return &GroupUserResource{client: &infisical.Client{Config: infisical.Config{
		HttpClient:            resty.New().SetBaseURL(server.URL),
		IsMachineIdentityAuth: true,
	}}}
}

func groupUserTestSchema(t *testing.T) schema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	NewGroupUserResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func groupUserCreatePlan(t *testing.T, s schema.Schema, groupID, username string) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: s}
	if diags := plan.Set(context.Background(), &GroupUserResourceModel{
		GroupID:  types.StringValue(groupID),
		Username: types.StringValue(username),
		UserID:   types.StringUnknown(),
	}); diags.HasError() {
		t.Fatal(diags)
	}
	return plan
}

func groupUserState(t *testing.T, s schema.Schema, model GroupUserResourceModel) tfsdk.State {
	t.Helper()
	state := tfsdk.State{Schema: s}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return state
}

func groupUserModel(t *testing.T, state tfsdk.State) GroupUserResourceModel {
	t.Helper()
	var model GroupUserResourceModel
	if diags := state.Get(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return model
}

func createGroupUser(t *testing.T, r *GroupUserResource, plan tfsdk.Plan) tfsdk.State {
	t.Helper()
	resp := resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema}}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	return resp.State
}

func readGroupUser(t *testing.T, r *GroupUserResource, state tfsdk.State) tfsdk.State {
	t.Helper()
	resp := resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	return resp.State
}

func deleteGroupUser(r *GroupUserResource, state tfsdk.State) resource.DeleteResponse {
	resp := resource.DeleteResponse{State: state}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, &resp)
	return resp
}

func TestGroupUserLifecycle(t *testing.T) {
	s := groupUserTestSchema(t)
	backend := newFakeGroupUserBackend(t)
	r := newGroupUserTestResource(t, backend)

	state := createGroupUser(t, r, groupUserCreatePlan(t, s, groupUserTestGroupID, "alice@example.com"))

	created := groupUserModel(t, state)
	if created.GroupID.ValueString() != groupUserTestGroupID || created.Username.ValueString() != "alice@example.com" || created.UserID.ValueString() != "u-alice" {
		t.Errorf("unexpected state after create: %+v", created)
	}
	if !backend.members[groupUserTestGroupID]["alice@example.com"] {
		t.Fatal("expected alice to be a member of the group on the backend")
	}
	if want := "POST /api/v1/groups/" + groupUserTestGroupID + "/users/alice@example.com"; backend.requests[0] != want {
		t.Errorf("expected the first request to be %q, got %q", want, backend.requests[0])
	}

	refreshed := groupUserModel(t, readGroupUser(t, r, state))
	if refreshed != created {
		t.Errorf("expected no drift after a refresh, got %+v, want %+v", refreshed, created)
	}

	if resp := deleteGroupUser(r, state); resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if backend.members[groupUserTestGroupID]["alice@example.com"] {
		t.Error("expected alice to be removed from the group after delete")
	}
}

func TestGroupUserReadDropsRemovedMembership(t *testing.T) {
	for name, remove := range map[string]func(b *fakeGroupUserBackend){
		"user removed outside Terraform": func(b *fakeGroupUserBackend) { delete(b.members[groupUserTestGroupID], "alice@example.com") },
		"group deleted":                  func(b *fakeGroupUserBackend) { delete(b.members, groupUserTestGroupID) },
	} {
		t.Run(name, func(t *testing.T) {
			s := groupUserTestSchema(t)
			backend := newFakeGroupUserBackend(t)
			r := newGroupUserTestResource(t, backend)

			state := createGroupUser(t, r, groupUserCreatePlan(t, s, groupUserTestGroupID, "alice@example.com"))
			remove(backend)

			if refreshed := readGroupUser(t, r, state); !refreshed.Raw.IsNull() {
				t.Error("expected the resource to be removed from state")
			}
		})
	}
}

func TestGroupUserReadEmptyID(t *testing.T) {
	backend := newFakeGroupUserBackend(t)
	r := newGroupUserTestResource(t, backend)

	state := groupUserState(t, groupUserTestSchema(t), GroupUserResourceModel{
		GroupID:  types.StringValue(""),
		Username: types.StringValue("alice@example.com"),
		UserID:   types.StringValue("u-alice"),
	})
	if refreshed := readGroupUser(t, r, state); !refreshed.Raw.IsNull() {
		t.Error("expected the resource to be removed from state")
	}
	if len(backend.requests) != 0 {
		t.Errorf("expected no API call, got %v", backend.requests)
	}
}

func TestGroupUserReadMatchesUsernameExactly(t *testing.T) {
	s := groupUserTestSchema(t)
	backend := newFakeGroupUserBackend(t)
	r := newGroupUserTestResource(t, backend)
	backend.members[groupUserTestGroupID]["bob@example.community"] = true

	notMember := groupUserState(t, s, GroupUserResourceModel{
		GroupID:  types.StringValue(groupUserTestGroupID),
		Username: types.StringValue("bob@example.com"),
		UserID:   types.StringValue("u-bob"),
	})
	if refreshed := readGroupUser(t, r, notMember); !refreshed.Raw.IsNull() {
		t.Error("bob@example.com is not a member; a substring match with bob@example.community must not count")
	}

	backend.members[groupUserTestGroupID]["bob@example.com"] = true
	if refreshed := groupUserModel(t, readGroupUser(t, r, notMember)); refreshed.UserID.ValueString() != "u-bob" {
		t.Errorf("expected bob@example.com to be found with its own user ID, got %+v", refreshed)
	}
}

func TestGroupUserDelete(t *testing.T) {
	s := groupUserTestSchema(t)

	for name, tc := range map[string]struct {
		prepare        func(b *fakeGroupUserBackend)
		wantError      bool
		wantDeleteCall bool
	}{
		"already not a member":        {prepare: func(b *fakeGroupUserBackend) { delete(b.members[groupUserTestGroupID], "alice@example.com") }},
		"group no longer exists":      {prepare: func(b *fakeGroupUserBackend) { delete(b.members, groupUserTestGroupID) }},
		"missing permission surfaces": {prepare: func(b *fakeGroupUserBackend) { b.forbidDelete = true }, wantError: true, wantDeleteCall: true},
	} {
		t.Run(name, func(t *testing.T) {
			backend := newFakeGroupUserBackend(t)
			r := newGroupUserTestResource(t, backend)

			state := createGroupUser(t, r, groupUserCreatePlan(t, s, groupUserTestGroupID, "alice@example.com"))
			tc.prepare(backend)

			resp := deleteGroupUser(r, state)
			if resp.Diagnostics.HasError() != tc.wantError {
				t.Errorf("wantError=%t, got diagnostics: %v", tc.wantError, resp.Diagnostics)
			}
			// Delete checks membership first, so a user who is already gone never reaches the endpoint that would answer 403.
			if got := backend.requestsMatching(http.MethodDelete) > 0; got != tc.wantDeleteCall {
				t.Errorf("wantDeleteCall=%t, requests: %v", tc.wantDeleteCall, backend.requests)
			}
		})
	}
}

func TestGroupUserImport(t *testing.T) {
	s := groupUserTestSchema(t)

	for name, tc := range map[string]struct {
		importID  string
		wantError bool
		wantGone  bool
	}{
		"group and username": {importID: groupUserTestGroupID + ",alice@example.com"},
		"missing comma":      {importID: groupUserTestGroupID, wantError: true},
		// The username keeps everything after the first comma; no such user exists, so Read drops it.
		"comma in username stays": {importID: groupUserTestGroupID + ",odd,name@example.com", wantGone: true},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			backend := newFakeGroupUserBackend(t)
			backend.members[groupUserTestGroupID]["alice@example.com"] = true
			r := newGroupUserTestResource(t, backend)

			resp := resource.ImportStateResponse{State: groupUserState(t, s, GroupUserResourceModel{})}
			r.ImportState(ctx, resource.ImportStateRequest{ID: tc.importID}, &resp)

			if tc.wantError {
				if !resp.Diagnostics.HasError() || !strings.Contains(fmt.Sprint(resp.Diagnostics), "Invalid import ID") {
					t.Errorf("expected an Invalid import ID error, got: %v", resp.Diagnostics)
				}
				return
			}
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}

			refreshed := readGroupUser(t, r, resp.State)
			if tc.wantGone {
				if !refreshed.Raw.IsNull() {
					t.Error("expected a non-member to be dropped by the Read after import")
				}
				return
			}

			groupID, username, _ := strings.Cut(tc.importID, ",")
			imported := groupUserModel(t, refreshed)
			if imported.GroupID.ValueString() != groupID || imported.Username.ValueString() != username || imported.UserID.ValueString() != backend.users[username] {
				t.Errorf("unexpected state after import: %+v", imported)
			}
		})
	}
}
