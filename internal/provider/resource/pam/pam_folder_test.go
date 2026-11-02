package resource

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func pamFolderPlan(t *testing.T, r *pamFolderResource, model pamFolderResourceModel) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: pamResourceSchema(t, r)}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return plan
}

func pamFolderModel(t *testing.T, state tfsdk.State) pamFolderResourceModel {
	t.Helper()
	var model pamFolderResourceModel
	if diags := state.Get(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return model
}

func TestPamFolderLifecycle(t *testing.T) {
	backend := newFakePamBackend(t)
	r := &pamFolderResource{}
	configurePamResource(t, r, newPamTestClient(t, backend))

	state := createPamResource(t, r, pamFolderPlan(t, r, pamFolderResourceModel{
		ID:          types.StringUnknown(),
		Name:        types.StringValue("linux-team"),
		Description: types.StringValue("Linux fleet"),
	}))
	created := pamFolderModel(t, state)
	if _, ok := backend.folders[created.ID.ValueString()]; !ok {
		t.Fatalf("expected folder %s on the backend", created.ID.ValueString())
	}

	refreshed := pamFolderModel(t, readPamResource(t, r, state))
	if refreshed != created {
		t.Errorf("expected a refresh to match the created state, got %+v, want %+v", refreshed, created)
	}

	// Removing the description has to clear it, not leave the stored one in place.
	state = updatePamResource(t, r, pamFolderPlan(t, r, pamFolderResourceModel{
		ID:          created.ID,
		Name:        types.StringValue("linux-fleet"),
		Description: types.StringNull(),
	}), state)
	patches := backend.requestsTo("PATCH", "/api/v1/pam/folders/"+created.ID.ValueString())
	if len(patches) != 1 {
		t.Fatalf("expected one update request, got %d", len(patches))
	}
	if description, sent := patches[0]["description"]; !sent || description != nil {
		t.Errorf("expected the update to send a null description, got %v (sent: %v)", description, sent)
	}
	updated := pamFolderModel(t, readPamResource(t, r, state))
	if updated.Name.ValueString() != "linux-fleet" || !updated.Description.IsNull() {
		t.Errorf("unexpected state after update: %+v", updated)
	}

	deletePamResource(t, r, state)
	if len(backend.folders) != 0 {
		t.Error("expected the folder to be deleted")
	}
	if !readPamResource(t, r, state).Raw.IsNull() {
		t.Error("expected a folder deleted outside of Terraform to be dropped from state")
	}
	// Deleting a folder that is already gone is not an error.
	deletePamResource(t, r, state)
}

func TestPamFolderImport(t *testing.T) {
	backend := newFakePamBackend(t)
	backend.folders["f1"] = map[string]any{"id": "f1", "name": "dbas", "description": nil}
	r := &pamFolderResource{}
	configurePamResource(t, r, newPamTestClient(t, backend))

	imported := pamFolderModel(t, importPamResource(t, r, pamResourceSchema(t, r), "f1"))
	if imported.ID.ValueString() != "f1" || imported.Name.ValueString() != "dbas" || !imported.Description.IsNull() {
		t.Errorf("unexpected imported state: %+v", imported)
	}
}
