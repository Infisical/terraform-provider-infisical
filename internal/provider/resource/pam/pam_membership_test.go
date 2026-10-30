package resource

import (
	"context"
	"strings"
	"testing"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	testPamGroupID    = "88888888-8888-8888-8888-888888888888"
	testPamIdentityID = "99999999-9999-9999-9999-999999999999"
	testPamAliceEmail = "alice@acme.com"
	testPamAliceID    = "11111111-1111-1111-1111-111111111111"
)

// pamMembershipPlan builds a create plan, with computed attributes unknown.
func pamMembershipPlan(t *testing.T, r *PamMembershipBaseResource, parentID, member, role string) tfsdk.Plan {
	t.Helper()
	ctx := context.Background()
	plan := emptyPamPlan(pamResourceSchema(t, r))
	set := func(name string, value any) {
		if diags := plan.SetAttribute(ctx, path.Root(name), value); diags.HasError() {
			t.Fatal(diags)
		}
	}

	set("id", types.StringUnknown())
	set("role", types.StringValue(role))
	if r.Scope != pamMembershipScopeProduct {
		set(r.parentAttribute(), types.StringValue(parentID))
	}
	if r.Kind == infisical.PamMemberKindUser {
		set("email", types.StringValue(member))
		set("user_id", types.StringUnknown())
	} else {
		set(r.memberAttribute(), types.StringValue(member))
	}
	return plan
}

func pamMembershipValuesOf(t *testing.T, r *PamMembershipBaseResource, state tfsdk.State) pamMembershipValues {
	t.Helper()
	values, diags := r.getValues(context.Background(), state)
	if diags.HasError() {
		t.Fatal(diags)
	}
	return values
}

func newPamMembershipTestResource(t *testing.T, newResource func() resource.Resource, backend *fakePamBackend) *PamMembershipBaseResource {
	t.Helper()
	created := newResource()
	r, ok := created.(*PamMembershipBaseResource)
	if !ok {
		t.Fatalf("expected a *PamMembershipBaseResource, got %T", created)
	}
	configurePamResource(t, r, newPamTestClient(t, backend))
	return r
}

func TestPamMembershipLifecycle(t *testing.T) {
	tests := []struct {
		newResource func() resource.Resource
		member      string
		memberID    string
		role        string
		newRole     string
	}{
		{NewPamUserResource, testPamAliceEmail, testPamAliceID, "member", "admin"},
		{NewPamGroupResource, testPamGroupID, testPamGroupID, "member", "admin"},
		{NewPamIdentityResource, testPamIdentityID, testPamIdentityID, "member", "admin"},
		{NewPamFolderUserResource, testPamAliceEmail, testPamAliceID, "connector", "operator"},
		{NewPamFolderGroupResource, testPamGroupID, testPamGroupID, "connector", "auditor"},
		{NewPamFolderIdentityResource, testPamIdentityID, testPamIdentityID, "connector", "admin"},
		{NewPamAccountUserResource, testPamAliceEmail, testPamAliceID, "connector", "operator"},
		{NewPamAccountGroupResource, testPamGroupID, testPamGroupID, "operator", "connector"},
		{NewPamAccountIdentityResource, testPamIdentityID, testPamIdentityID, "operator", "auditor"},
	}

	for _, test := range tests {
		backend := newFakePamBackend(t)
		r := newPamMembershipTestResource(t, test.newResource, backend)

		t.Run(r.ResourceTypeName, func(t *testing.T) {
			parentID, scope := "", "product"
			switch r.Scope {
			case pamMembershipScopeFolder:
				parentID, scope = testPamFolderID, "folders/"+testPamFolderID
				backend.folders[testPamFolderID] = map[string]any{"id": testPamFolderID, "name": "linux-team"}
			case pamMembershipScopeAccount:
				parentID, scope = "a1", "accounts/a1"
				backend.accounts["a1"] = &fakePamAccount{ID: "a1", Type: "ssh"}
			}
			kind := string(r.Kind)
			if r.Scope != pamMembershipScopeProduct {
				backend.addMember("product", kind, test.memberID, "member")
			}

			state := createPamResource(t, r, pamMembershipPlan(t, r, parentID, test.member, test.role))
			created := pamMembershipValuesOf(t, r, state)
			membership := backend.membersOf(scope, kind)[test.memberID]
			if membership == nil || membership.Role != test.role {
				t.Fatalf("expected %s to be a %s on the backend, got %+v", test.memberID, test.role, membership)
			}
			if created.ID.ValueString() != membership.ID || created.MemberID.ValueString() != test.memberID {
				t.Errorf("unexpected state after create: %+v", created)
			}

			refreshed := readPamResource(t, r, state)
			if !refreshed.Raw.Equal(state.Raw) {
				t.Errorf("expected no drift after a refresh.\ngot:  %v\nwant: %v", refreshed.Raw, state.Raw)
			}

			updatePlan := tfsdk.Plan(refreshed)
			if diags := updatePlan.SetAttribute(context.Background(), path.Root("role"), test.newRole); diags.HasError() {
				t.Fatal(diags)
			}
			state = updatePamResource(t, r, updatePlan, refreshed)
			if got := backend.membersOf(scope, kind)[test.memberID].Role; got != test.newRole {
				t.Errorf("expected the role to change to %s, got %s", test.newRole, got)
			}

			importID := test.member
			if parentID != "" {
				importID = parentID + ":" + test.member
			}
			imported := importPamResource(t, r, pamResourceSchema(t, r), importID)
			if !imported.Raw.Equal(readPamResource(t, r, state).Raw) {
				t.Errorf("expected the import to match the managed state.\ngot:  %v\nwant: %v", imported.Raw, state.Raw)
			}

			deletePamResource(t, r, state)
			if _, ok := backend.membersOf(scope, kind)[test.memberID]; ok {
				t.Error("expected the membership to be removed")
			}
			if !readPamResource(t, r, state).Raw.IsNull() {
				t.Error("expected a membership removed outside of Terraform to be dropped from state")
			}
			deletePamResource(t, r, state)
		})
	}
}

func createPamResourceError(t *testing.T, r resource.Resource, plan tfsdk.Plan) string {
	t.Helper()
	resp := resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema}}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected Create to fail")
	}
	return resp.Diagnostics.Errors()[0].Detail()
}

// The API skips existing members, so the provider has to refuse them rather than adopt them.
func TestPamUserRefusesAnExistingMember(t *testing.T) {
	backend := newFakePamBackend(t)
	backend.addMember("product", "users", testPamAliceID, "admin")
	r := newPamMembershipTestResource(t, NewPamUserResource, backend)

	detail := createPamResourceError(t, r, pamMembershipPlan(t, r, "", testPamAliceEmail, "member"))
	if !strings.Contains(detail, "terraform import infisical_pam_user.<name> "+testPamAliceEmail) {
		t.Errorf("expected the error to explain how to import the membership, got %q", detail)
	}
	if role := backend.membersOf("product", "users")[testPamAliceID].Role; role != "admin" {
		t.Errorf("expected the existing membership to be left alone, got role %s", role)
	}
}

func TestPamFolderMembershipsNeedAPamMember(t *testing.T) {
	backend := newFakePamBackend(t)
	backend.folders[testPamFolderID] = map[string]any{"id": testPamFolderID, "name": "linux-team"}

	users := newPamMembershipTestResource(t, NewPamFolderUserResource, backend)
	if detail := createPamResourceError(t, users, pamMembershipPlan(t, users, testPamFolderID, testPamAliceEmail, "connector")); !strings.Contains(detail, "Add the user to PAM first") {
		t.Errorf("expected the error to say the user isn't a PAM member, got %q", detail)
	}

	groups := newPamMembershipTestResource(t, NewPamFolderGroupResource, backend)
	if detail := createPamResourceError(t, groups, pamMembershipPlan(t, groups, testPamFolderID, testPamGroupID, "connector")); !strings.Contains(detail, "Must be a PAM product member first") {
		t.Errorf("expected the API's error, got %q", detail)
	}
}

func TestPamMembershipReadDropsDeletedFolder(t *testing.T) {
	backend := newFakePamBackend(t)
	backend.folders[testPamFolderID] = map[string]any{"id": testPamFolderID, "name": "linux-team"}
	backend.addMember("product", "groups", testPamGroupID, "member")
	r := newPamMembershipTestResource(t, NewPamFolderGroupResource, backend)

	state := createPamResource(t, r, pamMembershipPlan(t, r, testPamFolderID, testPamGroupID, "connector"))
	delete(backend.folders, testPamFolderID)
	if !readPamResource(t, r, state).Raw.IsNull() {
		t.Error("expected the membership of a deleted folder to be dropped from state")
	}
}

func TestPamMembershipImportFormat(t *testing.T) {
	r := newPamMembershipTestResource(t, NewPamFolderUserResource, newFakePamBackend(t))
	s := pamResourceSchema(t, r)

	resp := resource.ImportStateResponse{State: tfsdk.State{Schema: s}}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: testPamAliceEmail}, &resp)
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "<folder_id>:<email>") {
		t.Errorf("expected an error naming the import format, got %v", resp.Diagnostics)
	}
}
