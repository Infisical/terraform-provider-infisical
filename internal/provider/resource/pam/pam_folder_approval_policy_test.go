package resource

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const (
	testPamApproverUserID  = "11111111-1111-1111-1111-111111111111"
	testPamApproverGroupID = "33333333-3333-3333-3333-333333333333"
	testPamPolicyID        = "77777777-7777-7777-7777-777777777777"
)

func pamApprovers(t *testing.T, approvers ...pamApproverModel) types.Set {
	t.Helper()
	// Built from a non-nil slice, so no approvers give the empty set the schema defaults to.
	set, diags := types.SetValueFrom(context.Background(), pamApproverType, append([]pamApproverModel{}, approvers...))
	if diags.HasError() {
		t.Fatal(diags)
	}
	return set
}

func pamApprovalConstraints(t *testing.T, minimum, maximum string) types.Object {
	t.Helper()
	object, diags := types.ObjectValueFrom(context.Background(), pamApprovalConstraintsAttrTypes, pamApprovalConstraintsModel{
		AccessDuration: pamAccessDurationModel{Min: types.StringValue(minimum), Max: types.StringValue(maximum)},
	})
	if diags.HasError() {
		t.Fatal(diags)
	}
	return object
}

func pamApprovalPolicyPlan(t *testing.T, r *pamFolderApprovalPolicyResource, model pamFolderApprovalPolicyModel) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: pamResourceSchema(t, r)}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return plan
}

func pamApprovalPolicyState(t *testing.T, state tfsdk.State) pamFolderApprovalPolicyModel {
	t.Helper()
	var model pamFolderApprovalPolicyModel
	if diags := state.Get(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return model
}

func newPamApprovalPolicyTestResource(t *testing.T, backend *fakePamBackend) *pamFolderApprovalPolicyResource {
	t.Helper()
	backend.folders[testPamFolderID] = map[string]any{"id": testPamFolderID, "name": "prod"}
	r := &pamFolderApprovalPolicyResource{}
	configurePamResource(t, r, newPamTestClient(t, backend))
	return r
}

func TestPamFolderApprovalPolicyLifecycle(t *testing.T) {
	backend := newFakePamBackend(t)
	r := newPamApprovalPolicyTestResource(t, backend)

	user := pamApproverModel{Type: types.StringValue(pamApproverTypeUser), ID: types.StringValue(testPamApproverUserID)}
	group := pamApproverModel{Type: types.StringValue(pamApproverTypeGroup), ID: types.StringValue(testPamApproverGroupID)}
	model := pamFolderApprovalPolicyModel{
		ID:          types.StringUnknown(),
		FolderID:    types.StringValue(testPamFolderID),
		Name:        types.StringValue("prod-approvals"),
		Approvers:   pamApprovers(t, user, group),
		Bypassers:   pamApprovers(t),
		Constraints: pamApprovalConstraints(t, pamDefaultMinAccessDuration, pamDefaultMaxAccessDuration),
	}
	state := createPamResource(t, r, pamApprovalPolicyPlan(t, r, model))

	created := backend.requestsTo("POST", "/api/v1/approval-policies/pam-access")[0]
	if created["scope"] != pamApprovalPolicyScopeFolder || created["scopeId"] != testPamFolderID || created["name"] != "prod-approvals" {
		t.Errorf("unexpected policy sent: %v", created)
	}
	// The API rejects null for these lists.
	if conditions, ok := created["conditions"].([]any); !ok || len(conditions) != 0 {
		t.Errorf("expected an empty conditions list, got %v", created["conditions"])
	}
	if bypassers, ok := created["bypassers"].([]any); !ok || len(bypassers) != 0 {
		t.Errorf("expected an empty bypassers list, got %v", created["bypassers"])
	}
	accessDuration := jsonObject(t, jsonObject(t, created["constraints"])["accessDuration"])
	if accessDuration["min"] != "30s" || accessDuration["max"] != "7d" {
		t.Errorf("expected the default access duration, got %v", accessDuration)
	}
	steps, ok := created["steps"].([]any)
	if !ok || len(steps) != 1 {
		t.Fatalf("expected one step, got %v", created["steps"])
	}
	step := jsonObject(t, steps[0])
	if approvers, ok := step["approvers"].([]any); !ok || len(approvers) != 2 || step["requiredApprovals"] != float64(1) || step["notifyApprovers"] != true {
		t.Errorf("unexpected step sent: %v", step)
	}

	if refreshed := readPamResource(t, r, state); !refreshed.Raw.Equal(state.Raw) {
		t.Errorf("expected no drift after a refresh.\ngot:  %v\nwant: %v", refreshed.Raw, state.Raw)
	}

	if id := pamApprovalPolicyState(t, state).ID.ValueString(); id != testPamFolderID {
		t.Errorf("expected the folder ID as the resource ID, got %q", id)
	}

	// Rules added in Infisical survive an update, since the resource never sends them.
	var policyID string
	for id := range backend.approvalPolicies {
		policyID = id
	}
	rules := map[string]any{"inputs": []any{}, "rules": []any{map[string]any{"name": "jira-approved"}}}
	backend.approvalPolicies[policyID]["autoApproval"] = rules

	updated := model
	updated.ID = types.StringValue(testPamFolderID)
	updated.Bypassers = pamApprovers(t, user)
	updated.Constraints = pamApprovalConstraints(t, pamDefaultMinAccessDuration, "1d")
	state = updatePamResource(t, r, pamApprovalPolicyPlan(t, r, updated), state)
	policyPath := "/api/v1/approval-policies/pam-access/" + policyID
	patch := backend.requestsTo("PATCH", policyPath)[0]
	if _, sent := patch["autoApproval"]; sent {
		t.Error("expected the update to leave auto-approval rules out")
	}
	if kept, ok := jsonObject(t, backend.approvalPolicies[policyID]["autoApproval"])["rules"].([]any); !ok || len(kept) != 1 {
		t.Errorf("expected the stored rules to be kept, got %v", kept)
	}
	if bypassers, ok := patch["bypassers"].([]any); !ok || len(bypassers) != 1 {
		t.Errorf("expected one bypasser to be sent, got %v", patch["bypassers"])
	}

	// Removing the last bypasser has to clear the stored list.
	cleared := updated
	cleared.Bypassers = pamApprovers(t)
	state = updatePamResource(t, r, pamApprovalPolicyPlan(t, r, cleared), state)
	if bypassers, ok := backend.requestsTo("PATCH", policyPath)[1]["bypassers"].([]any); !ok || len(bypassers) != 0 {
		t.Errorf("expected an empty bypassers list to be sent, got %v", bypassers)
	}
	if refreshed := pamApprovalPolicyState(t, readPamResource(t, r, state)); len(refreshed.Bypassers.Elements()) != 0 {
		t.Errorf("expected no bypassers after the update, got %v", refreshed.Bypassers)
	}

	deletePamResource(t, r, state)
	if len(backend.approvalPolicies) != 0 {
		t.Error("expected the policy to be deleted")
	}
	if !readPamResource(t, r, state).Raw.IsNull() {
		t.Error("expected a policy deleted outside of Terraform to be dropped from state")
	}
	// Deleting a policy that is already gone is not an error.
	deletePamResource(t, r, state)
}

func TestPamFolderApprovalPolicyImport(t *testing.T) {
	backend := newFakePamBackend(t)
	folderID := testPamFolderID
	backend.approvalPolicies[testPamPolicyID] = map[string]any{
		"id": testPamPolicyID, "name": "PAM Folder Approval - dbas", "scopeType": pamApprovalPolicyScopeFolder, "scopeId": folderID,
		"steps": []any{map[string]any{
			"name": nil, "requiredApprovals": 1, "notifyApprovers": true,
			"approvers": []any{map[string]any{"type": "group", "id": testPamApproverGroupID}},
		}},
		"bypassers":   []any{map[string]any{"type": "user", "id": testPamApproverUserID}},
		"constraints": map[string]any{"version": 1, "constraints": map[string]any{"accessDuration": map[string]any{"min": "30s", "max": "7d"}}},
	}
	r := newPamApprovalPolicyTestResource(t, backend)

	imported := pamApprovalPolicyState(t, importPamResource(t, r, pamResourceSchema(t, r), folderID))
	if imported.ID.ValueString() != folderID || imported.FolderID.ValueString() != folderID || imported.Name.ValueString() != "PAM Folder Approval - dbas" || len(imported.Approvers.Elements()) != 1 {
		t.Errorf("unexpected imported policy: %+v", imported)
	}
	if len(imported.Bypassers.Elements()) != 1 {
		t.Errorf("expected one bypasser, got %v", imported.Bypassers)
	}
	accessDuration := objectAttributes(t, imported.Constraints.Attributes()["access_duration"])
	if stringAttribute(t, accessDuration, "max") != "7d" {
		t.Errorf("unexpected imported constraints: %v", imported.Constraints)
	}

	resp := resource.ImportStateResponse{State: tfsdk.State{Schema: pamResourceSchema(t, r)}}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: "prod-approvals"}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Error("expected an import ID that isn't a UUID to be rejected")
	}
}

func TestPamFolderApprovalPolicyReadDropsAnEmptyID(t *testing.T) {
	r := newPamApprovalPolicyTestResource(t, newFakePamBackend(t))
	s := pamResourceSchema(t, r)
	state := tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(context.Background()), nil)}
	if diags := state.SetAttribute(context.Background(), path.Root("id"), ""); diags.HasError() {
		t.Fatal(diags)
	}
	if !readPamResource(t, r, state).Raw.IsNull() {
		t.Error("expected a policy without an ID to be dropped from state")
	}
}
