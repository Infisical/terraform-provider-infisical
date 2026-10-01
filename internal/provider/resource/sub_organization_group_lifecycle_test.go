package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	testRootOrgID = "root-org"
	testSubOrgID  = "sub-org"
)

// Fake backend that answers like the real API, quirks included (a 400 for an already unlinked
// group, "already a member" for a duplicate link).
type fakeSubOrgGroupBackend struct {
	t            *testing.T
	sessionOrgID string
	rootGroups   map[string]infisical.OrgGroupMembershipGroup
	links        map[string]infisical.OrgGroupMembership
}

func newFakeSubOrgGroupBackend(t *testing.T) *fakeSubOrgGroupBackend {
	return &fakeSubOrgGroupBackend{
		t:            t,
		sessionOrgID: testSubOrgID,
		rootGroups: map[string]infisical.OrgGroupMembershipGroup{
			"g1": {ID: "g1", Name: "Platform", Slug: "platform", OrgID: testRootOrgID},
			"g2": {ID: "g2", Name: "Security", Slug: "security", OrgID: testRootOrgID},
		},
		links: map[string]infisical.OrgGroupMembership{},
	}
}

func (b *fakeSubOrgGroupBackend) writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		b.t.Error(err)
	}
}

func (b *fakeSubOrgGroupBackend) apiError(w http.ResponseWriter, status int, message string) {
	b.writeJSON(w, status, map[string]any{"statusCode": status, "message": message})
}

func (b *fakeSubOrgGroupBackend) decodeRoles(w http.ResponseWriter, r *http.Request) ([]infisical.OrgGroupMembershipRole, bool) {
	var body struct {
		Roles []infisical.OrgGroupMembershipRoleRequest `json:"roles"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		b.apiError(w, http.StatusUnprocessableEntity, err.Error())
		return nil, false
	}

	roles := make([]infisical.OrgGroupMembershipRole, 0, len(body.Roles))
	for _, role := range body.Roles {
		apiRole := infisical.OrgGroupMembershipRole{Role: role.Role, IsTemporary: role.IsTemporary}
		if role.IsTemporary {
			// Mirrors the API's z.string().datetime(), which only takes UTC.
			if role.TemporaryAccessStartTime == nil || role.TemporaryAccessStartTime.Location() != time.UTC {
				b.apiError(w, http.StatusUnprocessableEntity, "Invalid datetime")
				return nil, false
			}
			apiRole.TemporaryRange = &role.TemporaryRange
			apiRole.TemporaryAccessStartTime = role.TemporaryAccessStartTime
		}
		roles = append(roles, apiRole)
	}
	return roles, true
}

func (b *fakeSubOrgGroupBackend) handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/identities/details", func(w http.ResponseWriter, _ *http.Request) {
		b.writeJSON(w, http.StatusOK, map[string]any{"identityDetails": map[string]any{
			"organization": map[string]string{"id": b.sessionOrgID, "name": b.sessionOrgID, "slug": b.sessionOrgID},
		}})
	})

	mux.HandleFunc("GET /api/v1/organization/available-groups", func(w http.ResponseWriter, _ *http.Request) {
		groups := []infisical.AvailableGroup{}
		if b.sessionOrgID != testRootOrgID {
			for id, group := range b.rootGroups {
				if _, linked := b.links[id]; !linked {
					groups = append(groups, infisical.AvailableGroup{ID: group.ID, Name: group.Name, Slug: group.Slug})
				}
			}
		}
		b.writeJSON(w, http.StatusOK, infisical.ListAvailableGroupsResponse{Groups: groups})
	})

	mux.HandleFunc("GET /api/v1/organizations/memberships/groups", func(w http.ResponseWriter, _ *http.Request) {
		memberships := []infisical.OrgGroupMembership{}
		for _, link := range b.links {
			memberships = append(memberships, link)
		}
		b.writeJSON(w, http.StatusOK, infisical.ListOrgGroupMembershipsResponse{GroupMemberships: memberships, TotalCount: len(memberships)})
	})

	mux.HandleFunc("POST /api/v1/organizations/memberships/groups/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		group, ok := b.rootGroups[id]
		if !ok {
			b.apiError(w, http.StatusBadRequest, "Only groups from parent organization can be linked to this sub-organization")
			return
		}
		if _, linked := b.links[id]; linked {
			b.apiError(w, http.StatusBadRequest, "Group is already a member")
			return
		}
		roles, ok := b.decodeRoles(w, r)
		if !ok {
			return
		}
		b.links[id] = infisical.OrgGroupMembership{ID: "m-" + id, GroupID: id, Group: group, Roles: roles}
		b.writeJSON(w, http.StatusOK, infisical.OrgGroupMembershipResponse{GroupMembership: b.links[id]})
	})

	mux.HandleFunc("GET /api/v1/organizations/memberships/groups/{id}", func(w http.ResponseWriter, r *http.Request) {
		link, ok := b.links[r.PathValue("id")]
		if !ok {
			b.apiError(w, http.StatusNotFound, "Group not found")
			return
		}
		b.writeJSON(w, http.StatusOK, infisical.OrgGroupMembershipResponse{GroupMembership: link})
	})

	mux.HandleFunc("PATCH /api/v1/organizations/memberships/groups/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		link, ok := b.links[id]
		if !ok {
			b.apiError(w, http.StatusNotFound, "Group not found")
			return
		}
		roles, ok := b.decodeRoles(w, r)
		if !ok {
			return
		}
		link.Roles = roles
		b.links[id] = link
		b.writeJSON(w, http.StatusOK, infisical.OrgGroupMembershipResponse{GroupMembership: link})
	})

	mux.HandleFunc("DELETE /api/v1/organizations/memberships/groups/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if _, exists := b.rootGroups[id]; !exists {
			b.apiError(w, http.StatusNotFound, "Group not found")
			return
		}
		link, linked := b.links[id]
		if !linked {
			b.apiError(w, http.StatusBadRequest, "Group doesn't have membership")
			return
		}
		delete(b.links, id)
		b.writeJSON(w, http.StatusOK, map[string]any{"groupMembership": map[string]string{"id": link.ID, "groupId": id}})
	})

	return mux
}

func newSubOrgGroupTestResource(t *testing.T, backend *fakeSubOrgGroupBackend) *subOrganizationGroupResource {
	t.Helper()
	server := httptest.NewServer(backend.handler())
	t.Cleanup(server.Close)
	return &subOrganizationGroupResource{client: &infisical.Client{Config: infisical.Config{
		HttpClient:            resty.New().SetBaseURL(server.URL),
		IsMachineIdentityAuth: true,
	}}}
}

func subOrgGroupTestSchema(t *testing.T) schema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	NewSubOrganizationGroupResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func memberRole() subOrganizationGroupRole {
	return subOrganizationGroupRole{
		RoleSlug:                 types.StringValue("member"),
		IsTemporary:              types.BoolValue(false),
		TemporaryRange:           types.StringNull(),
		TemporaryAccessStartTime: types.StringNull(),
	}
}

// What the plan looks like on create: computed attributes are still unknown.
func subOrgGroupCreatePlan(t *testing.T, s schema.Schema, groupID, groupSlug types.String, roles ...subOrganizationGroupRole) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: s}
	if diags := plan.Set(context.Background(), &subOrganizationGroupResourceModel{
		ID:        types.StringUnknown(),
		GroupID:   groupID,
		GroupSlug: groupSlug,
		GroupName: types.StringUnknown(),
		Roles:     roles,
	}); diags.HasError() {
		t.Fatal(diags)
	}
	return plan
}

func subOrgGroupModel(t *testing.T, state tfsdk.State) subOrganizationGroupResourceModel {
	t.Helper()
	var model subOrganizationGroupResourceModel
	if diags := state.Get(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return model
}

func createSubOrgGroup(t *testing.T, r *subOrganizationGroupResource, plan tfsdk.Plan) tfsdk.State {
	t.Helper()
	resp := resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema}}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	return resp.State
}

func readSubOrgGroup(t *testing.T, r *subOrganizationGroupResource, state tfsdk.State) tfsdk.State {
	t.Helper()
	resp := resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	return resp.State
}

func TestSubOrganizationGroupLifecycle(t *testing.T) {
	ctx := context.Background()
	s := subOrgGroupTestSchema(t)
	backend := newFakeSubOrgGroupBackend(t)
	r := newSubOrgGroupTestResource(t, backend)

	// Linking by slug resolves the group ID from the available groups.
	state := createSubOrgGroup(t, r, subOrgGroupCreatePlan(t, s, types.StringUnknown(), types.StringValue("platform"), memberRole()))
	created := subOrgGroupModel(t, state)
	if created.GroupID.ValueString() != "g1" || created.ID.ValueString() != "m-g1" || created.GroupName.ValueString() != "Platform" {
		t.Errorf("unexpected state after create: %+v", created)
	}
	if _, linked := backend.links["g1"]; !linked {
		t.Fatal("expected g1 to be linked on the backend")
	}

	// Update swaps in a temporary role written with an offset, which must reach the API as UTC.
	updatePlan := tfsdk.Plan{Schema: s, Raw: state.Raw}
	if diags := updatePlan.SetAttribute(ctx, path.Root("roles"), []subOrganizationGroupRole{
		memberRole(),
		{
			RoleSlug:                 types.StringValue("admin"),
			IsTemporary:              types.BoolValue(true),
			TemporaryRange:           types.StringNull(),
			TemporaryAccessStartTime: types.StringValue("2026-10-01T11:00:00+02:00"),
		},
	}); diags.HasError() {
		t.Fatal(diags)
	}
	updateResp := resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{Plan: updatePlan, State: state}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatal(updateResp.Diagnostics)
	}
	if got := len(backend.links["g1"].Roles); got != 2 {
		t.Fatalf("expected 2 roles on the backend, got %d", got)
	}

	// Refreshing right after the update must not show drift.
	refreshed := subOrgGroupModel(t, readSubOrgGroup(t, r, updateResp.State))
	if !refreshed.Roles[1].TemporaryRange.IsNull() || refreshed.Roles[1].TemporaryAccessStartTime.ValueString() != "2026-10-01T11:00:00+02:00" {
		t.Errorf("expected the configured form to survive a refresh, got %+v", refreshed.Roles[1])
	}

	deleteResp := resource.DeleteResponse{State: updateResp.State}
	r.Delete(ctx, resource.DeleteRequest{State: updateResp.State}, &deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatal(deleteResp.Diagnostics)
	}
	if _, linked := backend.links["g1"]; linked {
		t.Error("expected g1 to be unlinked after delete")
	}
}

// Someone unlinks the group in the UI: refresh drops it so the next plan re-creates the link.
func TestSubOrganizationGroupReadDropsExternallyUnlinkedGroup(t *testing.T) {
	s := subOrgGroupTestSchema(t)
	backend := newFakeSubOrgGroupBackend(t)
	r := newSubOrgGroupTestResource(t, backend)

	state := createSubOrgGroup(t, r, subOrgGroupCreatePlan(t, s, types.StringValue("g1"), types.StringUnknown(), memberRole()))
	delete(backend.links, "g1")

	if refreshed := readSubOrgGroup(t, r, state); !refreshed.Raw.IsNull() {
		t.Error("expected the resource to be removed from state")
	}
}

func TestSubOrganizationGroupDelete(t *testing.T) {
	s := subOrgGroupTestSchema(t)

	for name, tc := range map[string]struct {
		groupID   string
		unlink    bool
		wantError bool
	}{
		"already unlinked answers 400": {groupID: "g1", unlink: true},
		"group no longer exists":       {groupID: "gone"},
		"still linked after a failure": {groupID: "g1", wantError: true},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			backend := newFakeSubOrgGroupBackend(t)
			r := newSubOrgGroupTestResource(t, backend)

			state := createSubOrgGroup(t, r, subOrgGroupCreatePlan(t, s, types.StringValue("g1"), types.StringUnknown(), memberRole()))
			if diags := state.SetAttribute(ctx, path.Root("group_id"), tc.groupID); diags.HasError() {
				t.Fatal(diags)
			}
			if tc.unlink {
				delete(backend.links, "g1")
			}
			if tc.wantError {
				// Make the DELETE fail while the link is still there.
				r = &subOrganizationGroupResource{client: failingDeleteClient(t, backend)}
			}

			resp := resource.DeleteResponse{State: state}
			r.Delete(ctx, resource.DeleteRequest{State: state}, &resp)
			if resp.Diagnostics.HasError() != tc.wantError {
				t.Errorf("wantError=%t, got diagnostics: %v", tc.wantError, resp.Diagnostics)
			}
		})
	}
}

// Serves the fake backend but fails every DELETE with a 403.
func failingDeleteClient(t *testing.T, backend *fakeSubOrgGroupBackend) *infisical.Client {
	t.Helper()
	inner := backend.handler()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			backend.apiError(w, http.StatusForbidden, "forbidden")
			return
		}
		inner.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	return &infisical.Client{Config: infisical.Config{
		HttpClient:            resty.New().SetBaseURL(server.URL),
		IsMachineIdentityAuth: true,
	}}
}

func TestSubOrganizationGroupImport(t *testing.T) {
	s := subOrgGroupTestSchema(t)

	for name, tc := range map[string]struct {
		importID     string
		sessionOrgID string
		wantGroupID  string
		wantError    string
	}{
		"by id":                    {importID: "00000000-0000-4000-8000-000000000001", wantGroupID: "00000000-0000-4000-8000-000000000001"},
		"by slug":                  {importID: "platform", wantGroupID: "00000000-0000-4000-8000-000000000001"},
		"unknown slug":             {importID: "nope", wantError: "Group not found"},
		"native group in root org": {importID: "platform", sessionOrgID: testRootOrgID, wantError: "belongs to the organization the provider is scoped to"},
		"unknown id":               {importID: "00000000-0000-4000-8000-000000000009", wantError: "Group not found"},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			backend := newFakeSubOrgGroupBackend(t)
			const linkedID = "00000000-0000-4000-8000-000000000001"
			backend.rootGroups = map[string]infisical.OrgGroupMembershipGroup{
				linkedID: {ID: linkedID, Name: "Platform", Slug: "platform", OrgID: testRootOrgID},
			}
			backend.links[linkedID] = infisical.OrgGroupMembership{ID: "m-1", GroupID: linkedID, Group: backend.rootGroups[linkedID]}
			if tc.sessionOrgID != "" {
				backend.sessionOrgID = tc.sessionOrgID
			}
			r := newSubOrgGroupTestResource(t, backend)

			importState := tfsdk.State{Schema: s}
			if diags := importState.Set(ctx, &subOrganizationGroupResourceModel{}); diags.HasError() {
				t.Fatal(diags)
			}
			resp := resource.ImportStateResponse{State: importState}
			r.ImportState(ctx, resource.ImportStateRequest{ID: tc.importID}, &resp)

			if tc.wantError != "" {
				if !resp.Diagnostics.HasError() || !strings.Contains(fmt.Sprint(resp.Diagnostics), tc.wantError) {
					t.Errorf("expected an error containing %q, got: %v", tc.wantError, resp.Diagnostics)
				}
				return
			}
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}

			// Imports only set group_id, so Read has to fill in the rest.
			imported := subOrgGroupModel(t, readSubOrgGroup(t, r, resp.State))
			if imported.GroupID.ValueString() != tc.wantGroupID || imported.GroupSlug.ValueString() != "platform" || imported.ID.ValueString() != "m-1" {
				t.Errorf("unexpected state after import: %+v", imported)
			}
		})
	}
}

func TestSubOrganizationGroupCreateErrors(t *testing.T) {
	s := subOrgGroupTestSchema(t)

	for name, tc := range map[string]struct {
		groupID      types.String
		groupSlug    types.String
		sessionOrgID string
		preLinked    bool
		wantError    string
	}{
		"already linked by slug": {groupID: types.StringUnknown(), groupSlug: types.StringValue("platform"), preLinked: true, wantError: "terraform import"},
		"already linked by id":   {groupID: types.StringValue("g1"), groupSlug: types.StringUnknown(), preLinked: true, wantError: "terraform import"},
		"unknown slug":           {groupID: types.StringUnknown(), groupSlug: types.StringValue("nope"), wantError: "Group not found"},
		"root scoped by slug":    {groupID: types.StringUnknown(), groupSlug: types.StringValue("platform"), sessionOrgID: testRootOrgID, preLinked: true, wantError: "auth.organization_slug"},
		"root scoped by id":      {groupID: types.StringValue("g1"), groupSlug: types.StringUnknown(), sessionOrgID: testRootOrgID, preLinked: true, wantError: "auth.organization_slug"},
	} {
		t.Run(name, func(t *testing.T) {
			backend := newFakeSubOrgGroupBackend(t)
			if tc.sessionOrgID != "" {
				backend.sessionOrgID = tc.sessionOrgID
			}
			if tc.preLinked {
				// In the root org every root group already has a native membership.
				backend.links["g1"] = infisical.OrgGroupMembership{ID: "m-g1", GroupID: "g1", Group: backend.rootGroups["g1"]}
			}
			r := newSubOrgGroupTestResource(t, backend)

			resp := resource.CreateResponse{State: tfsdk.State{Schema: s}}
			r.Create(context.Background(), resource.CreateRequest{Plan: subOrgGroupCreatePlan(t, s, tc.groupID, tc.groupSlug, memberRole())}, &resp)
			if !resp.Diagnostics.HasError() || !strings.Contains(fmt.Sprint(resp.Diagnostics), tc.wantError) {
				t.Errorf("expected an error containing %q, got: %v", tc.wantError, resp.Diagnostics)
			}
		})
	}
}
