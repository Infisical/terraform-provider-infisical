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
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

type fakeSecretImport struct {
	ProjectID, Environment, SecretPath     string
	SourceProjectID, ImportEnv, ImportPath string
	IsReplication                          bool
}

// Fake of the v2 secret import routes, following backend/src/server/routes/v2/secret-import-router.ts and
// createImport, getImportById and deleteImport in backend/src/services/secret-import/secret-import-service.ts.
type fakeSecretImportBackend struct {
	t       *testing.T
	imports map[string]fakeSecretImport
	nextID  int
	gets    int

	// ignoresSourceProject is Infisical before v0.161.12: the create body schema strips sourceProjectId.
	ignoresSourceProject bool
	// omitsImportProjectID is Infisical before v0.167.16: get by ID strips importEnv.projectId.
	omitsImportProjectID bool
	failDelete           bool
}

func newFakeSecretImportBackend(t *testing.T) *fakeSecretImportBackend {
	return &fakeSecretImportBackend{t: t, imports: map[string]fakeSecretImport{}}
}

func (b *fakeSecretImportBackend) writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		b.t.Error(err)
	}
}

func (b *fakeSecretImportBackend) notFound(w http.ResponseWriter) {
	b.writeJSON(w, http.StatusNotFound, map[string]any{"statusCode": http.StatusNotFound, "message": "Secret import not found", "error": "NotFound"})
}

// canonicalizeSecretPath in Infisical core: create stores canonical paths and folder lookups skip empty segments.
func fakeCanonicalPath(p string) string {
	return "/" + strings.Join(strings.FieldsFunc(p, func(r rune) bool { return r == '/' }), "/")
}

func (b *fakeSecretImportBackend) handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v2/secret-imports", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ProjectID     string `json:"projectId"`
			Environment   string `json:"environment"`
			Path          string `json:"path"`
			IsReplication bool   `json:"isReplication"`
			Import        struct {
				SourceProjectID string `json:"sourceProjectId"`
				Environment     string `json:"environment"`
				Path            string `json:"path"`
			} `json:"import"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			b.writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"statusCode": http.StatusUnprocessableEntity, "message": err.Error()})
			return
		}
		sourceProjectID := body.Import.SourceProjectID
		if sourceProjectID == "" || b.ignoresSourceProject {
			sourceProjectID = body.ProjectID
		}
		b.nextID++
		id := fmt.Sprintf("00000000-0000-4000-8000-%012d", b.nextID)
		b.imports[id] = fakeSecretImport{
			ProjectID: body.ProjectID, Environment: body.Environment, SecretPath: fakeCanonicalPath(body.Path),
			SourceProjectID: sourceProjectID, ImportEnv: body.Import.Environment, ImportPath: fakeCanonicalPath(body.Import.Path),
			IsReplication: body.IsReplication,
		}
		b.writeJSON(w, http.StatusOK, map[string]any{"secretImport": map[string]any{"id": id}})
	})

	// getImportById only checks Read on the destination folder, so it ignores the cross-project sharing setting.
	mux.HandleFunc("GET /api/v2/secret-imports/{id}", func(w http.ResponseWriter, r *http.Request) {
		b.gets++
		imp, ok := b.imports[r.PathValue("id")]
		if !ok {
			b.notFound(w)
			return
		}
		importEnv := map[string]string{"slug": imp.ImportEnv, "projectId": imp.SourceProjectID}
		if b.omitsImportProjectID {
			delete(importEnv, "projectId")
		}
		b.writeJSON(w, http.StatusOK, map[string]any{"secretImport": map[string]any{
			"id": r.PathValue("id"), "projectId": imp.ProjectID, "secretPath": imp.SecretPath, "importPath": imp.ImportPath,
			"isReplication": imp.IsReplication,
			"environment":   map[string]string{"slug": imp.Environment},
			"importEnv":     importEnv,
		}})
	})

	// deleteImport finds the import in the destination folder.
	mux.HandleFunc("DELETE /api/v2/secret-imports/{id}", func(w http.ResponseWriter, r *http.Request) {
		if b.failDelete {
			b.writeJSON(w, http.StatusInternalServerError, map[string]any{"statusCode": http.StatusInternalServerError, "message": "Something went wrong"})
			return
		}
		var body struct {
			ProjectID   string `json:"projectId"`
			Environment string `json:"environment"`
			Path        string `json:"path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			b.writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"statusCode": http.StatusUnprocessableEntity, "message": err.Error()})
			return
		}
		id := r.PathValue("id")
		if imp, ok := b.imports[id]; !ok || imp.ProjectID != body.ProjectID || imp.Environment != body.Environment || imp.SecretPath != fakeCanonicalPath(body.Path) {
			b.notFound(w)
			return
		}
		delete(b.imports, id)
		b.writeJSON(w, http.StatusOK, map[string]any{"secretImport": map[string]any{"id": id}})
	})

	return mux
}

func newSecretImportTestResource(t *testing.T, backend *fakeSecretImportBackend) *projectSecretImportResource {
	t.Helper()
	server := httptest.NewServer(backend.handler())
	t.Cleanup(server.Close)
	return &projectSecretImportResource{client: &infisical.Client{Config: infisical.Config{
		HttpClient:            resty.New().SetBaseURL(server.URL),
		IsMachineIdentityAuth: true,
	}}}
}

func secretImportTestSchema(t *testing.T) schema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	NewProjectSecretImportResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func secretImportTestModel(projectID, importProjectID types.String, isReplication bool) projectSecretImportResourceModel {
	return projectSecretImportResourceModel{
		ID:              types.StringUnknown(),
		ProjectID:       projectID,
		EnvironmentSlug: types.StringValue("prod"),
		SecretPath:      types.StringValue("/app"),
		ImportPath:      types.StringValue("/shared"),
		ImportEnvSlug:   types.StringValue("dev"),
		ImportProjectID: importProjectID,
		IsReplication:   types.BoolValue(isReplication),
	}
}

func secretImportTestPlan(t *testing.T, s schema.Schema, model projectSecretImportResourceModel) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: s}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return plan
}

func createSecretImportResp(t *testing.T, r *projectSecretImportResource, s schema.Schema, model projectSecretImportResourceModel) resource.CreateResponse {
	t.Helper()
	resp := resource.CreateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(context.Background()), nil)}}
	r.Create(context.Background(), resource.CreateRequest{Plan: secretImportTestPlan(t, s, model)}, &resp)
	return resp
}

func createSecretImport(t *testing.T, r *projectSecretImportResource, s schema.Schema, model projectSecretImportResourceModel) tfsdk.State {
	t.Helper()
	resp := createSecretImportResp(t, r, s, model)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	return resp.State
}

// readSecretImport returns the state after Read. Its Raw is null when Read removed the import.
func readSecretImport(t *testing.T, r *projectSecretImportResource, state tfsdk.State) tfsdk.State {
	t.Helper()
	resp := resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	return resp.State
}

func secretImportStateModel(t *testing.T, state tfsdk.State) projectSecretImportResourceModel {
	t.Helper()
	var model projectSecretImportResourceModel
	if diags := state.Get(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return model
}

func deleteSecretImport(t *testing.T, r *projectSecretImportResource, state tfsdk.State) {
	t.Helper()
	resp := resource.DeleteResponse{State: state}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
}

// importSecretImport runs ImportState and then Read, the way terraform import does.
func importSecretImport(t *testing.T, r *projectSecretImportResource, s schema.Schema, id string) tfsdk.State {
	t.Helper()
	resp := resource.ImportStateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(context.Background()), nil)}}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: id}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	return readSecretImport(t, r, resp.State)
}

func TestSecretImportCrossProjectLifecycle(t *testing.T) {
	backend := newFakeSecretImportBackend(t)
	r := newSecretImportTestResource(t, backend)
	s := secretImportTestSchema(t)

	state := createSecretImport(t, r, s, secretImportTestModel(types.StringValue("destination"), types.StringValue("source"), false))
	created := secretImportStateModel(t, state)
	id := created.ID.ValueString()
	want := fakeSecretImport{
		ProjectID: "destination", Environment: "prod", SecretPath: "/app",
		SourceProjectID: "source", ImportEnv: "dev", ImportPath: "/shared",
	}
	if got := backend.imports[id]; got != want {
		t.Fatalf("backend import = %+v, want %+v", got, want)
	}

	state = readSecretImport(t, r, state)
	if got := secretImportStateModel(t, state); got != created {
		t.Errorf("state after refresh = %+v, want %+v", got, created)
	}

	deleteSecretImport(t, r, state)
	if _, ok := backend.imports[id]; ok {
		t.Fatal("expected Delete to remove the import")
	}
	if !readSecretImport(t, r, state).Raw.IsNull() {
		t.Error("expected Read to remove the deleted import from state")
	}

	// Delete treats a 404 as success.
	deleteSecretImport(t, r, state)
}

// Existing configs have no import_project_id. The API returns the destination project as the source, which must
// leave import_project_id null.
func TestSecretImportSameProjectLifecycle(t *testing.T) {
	backend := newFakeSecretImportBackend(t)
	r := newSecretImportTestResource(t, backend)

	state := createSecretImport(t, r, secretImportTestSchema(t), secretImportTestModel(types.StringValue("destination"), types.StringNull(), false))
	if backend.gets != 0 {
		t.Errorf("same-project create made %d get by ID calls, want 0", backend.gets)
	}

	refreshed := readSecretImport(t, r, state)
	if refreshed.Raw.IsNull() {
		t.Fatal("expected Read to keep the import in state")
	}
	if got, want := secretImportStateModel(t, refreshed), secretImportStateModel(t, state); got != want {
		t.Errorf("state after refresh = %+v, want %+v", got, want)
	}

	deleteSecretImport(t, r, refreshed)
	if !readSecretImport(t, r, refreshed).Raw.IsNull() {
		t.Error("expected Read to remove the deleted import from state")
	}
}

// Delete finds the import by its destination folder, so Read must pick up a folder renamed outside Terraform, or
// destroy would get a 404 for the old folder, treat it as success, and leave the import behind.
func TestSecretImportDestinationRenamedOutsideTerraform(t *testing.T) {
	backend := newFakeSecretImportBackend(t)
	r := newSecretImportTestResource(t, backend)

	state := createSecretImport(t, r, secretImportTestSchema(t), secretImportTestModel(types.StringValue("destination"), types.StringValue("source"), false))
	id := secretImportStateModel(t, state).ID.ValueString()
	renamed := backend.imports[id]
	renamed.Environment, renamed.SecretPath = "staging", "/renamed"
	backend.imports[id] = renamed

	state = readSecretImport(t, r, state)
	if got := secretImportStateModel(t, state); got.EnvironmentSlug.ValueString() != "staging" || got.SecretPath.ValueString() != "/renamed" {
		t.Errorf("destination after refresh = %q %q, want staging /renamed", got.EnvironmentSlug.ValueString(), got.SecretPath.ValueString())
	}

	deleteSecretImport(t, r, state)
	if _, ok := backend.imports[id]; ok {
		t.Error("expected Delete to remove the import from its renamed folder")
	}
}

// The API stores canonical paths. A configured "/app/" must stay in state, or the import would be replaced on every
// apply.
func TestSecretImportReadKeepsEquivalentPath(t *testing.T) {
	backend := newFakeSecretImportBackend(t)
	r := newSecretImportTestResource(t, backend)

	model := secretImportTestModel(types.StringValue("destination"), types.StringNull(), false)
	model.SecretPath = types.StringValue("/app/")
	state := createSecretImport(t, r, secretImportTestSchema(t), model)

	if got := secretImportStateModel(t, readSecretImport(t, r, state)).SecretPath.ValueString(); got != "/app/" {
		t.Errorf("folder_path after refresh = %q, want /app/", got)
	}
}

func TestSecretImportReadEmptyID(t *testing.T) {
	backend := newFakeSecretImportBackend(t)
	r := newSecretImportTestResource(t, backend)
	s := secretImportTestSchema(t)

	model := secretImportTestModel(types.StringValue("destination"), types.StringNull(), false)
	model.ID = types.StringValue("")
	state := tfsdk.State{Schema: s}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}

	if !readSecretImport(t, r, state).Raw.IsNull() {
		t.Error("expected Read to remove an import with an empty ID")
	}
	if backend.gets != 0 {
		t.Errorf("Read with an empty ID made %d get by ID calls, want 0", backend.gets)
	}
}

func TestSecretImportImportState(t *testing.T) {
	backend := newFakeSecretImportBackend(t)
	r := newSecretImportTestResource(t, backend)
	s := secretImportTestSchema(t)

	for name, importProjectID := range map[string]types.String{
		"same project":  types.StringNull(),
		"cross-project": types.StringValue("source"),
	} {
		t.Run(name, func(t *testing.T) {
			created := secretImportStateModel(t, createSecretImport(t, r, s, secretImportTestModel(types.StringValue("destination"), importProjectID, false)))

			if got := secretImportStateModel(t, importSecretImport(t, r, s, created.ID.ValueString())); got != created {
				t.Errorf("imported state = %+v, want %+v", got, created)
			}
		})
	}
}

// A cross-project import on Infisical before v0.167.16 must fail and leave nothing behind.
func TestSecretImportCrossProjectOnOldServer(t *testing.T) {
	for name, backend := range map[string]*fakeSecretImportBackend{
		"before v0.161.12": {ignoresSourceProject: true, omitsImportProjectID: true},
		"before v0.167.16": {omitsImportProjectID: true},
	} {
		t.Run(name, func(t *testing.T) {
			backend.t, backend.imports = t, map[string]fakeSecretImport{}
			r := newSecretImportTestResource(t, backend)

			resp := createSecretImportResp(t, r, secretImportTestSchema(t), secretImportTestModel(types.StringValue("destination"), types.StringValue("source"), false))
			if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "v0.167.16") {
				t.Fatalf("expected an error that names v0.167.16, got %v", resp.Diagnostics)
			}
			if len(backend.imports) != 0 {
				t.Errorf("expected Create to delete the import again, backend has %+v", backend.imports)
			}
			if !resp.State.Raw.IsNull() {
				t.Error("expected no state after the import was deleted again")
			}
		})
	}
}

// When the cleanup delete fails, the import still exists and must stay in state so it isn't lost.
func TestSecretImportCrossProjectOnOldServerKeepsImportWhenDeleteFails(t *testing.T) {
	backend := newFakeSecretImportBackend(t)
	backend.omitsImportProjectID, backend.failDelete = true, true
	r := newSecretImportTestResource(t, backend)

	resp := createSecretImportResp(t, r, secretImportTestSchema(t), secretImportTestModel(types.StringValue("destination"), types.StringValue("source"), false))
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error")
	}
	var id types.String
	if diags := resp.State.GetAttribute(context.Background(), path.Root("id"), &id); diags.HasError() {
		t.Fatal(diags)
	}
	if _, ok := backend.imports[id.ValueString()]; !ok {
		t.Errorf("state ID %q doesn't match the import left in the backend %+v", id.ValueString(), backend.imports)
	}
}

// Infisical before v0.167.16 leaves importEnv.projectId out, so Read keeps import_project_id from state.
func TestSecretImportReadKeepsImportProjectIDOnOldServer(t *testing.T) {
	backend := newFakeSecretImportBackend(t)
	r := newSecretImportTestResource(t, backend)

	state := createSecretImport(t, r, secretImportTestSchema(t), secretImportTestModel(types.StringValue("destination"), types.StringValue("source"), false))
	backend.omitsImportProjectID = true

	if got := secretImportStateModel(t, readSecretImport(t, r, state)).ImportProjectID.ValueString(); got != "source" {
		t.Errorf("import_project_id after refresh = %q, want source", got)
	}
}

func TestSecretImportValidateConfig(t *testing.T) {
	s := secretImportTestSchema(t)

	for name, tc := range map[string]struct {
		projectID, importProjectID types.String
		isReplication, wantError   bool
	}{
		"cross-project":                          {types.StringValue("destination"), types.StringValue("source"), false, false},
		"cross-project replication":              {types.StringValue("destination"), types.StringValue("source"), true, true},
		"same project replication":               {types.StringValue("destination"), types.StringNull(), true, false},
		"import_project_id is project_id":        {types.StringValue("destination"), types.StringValue("destination"), false, true},
		"import_project_id not known yet":        {types.StringValue("destination"), types.StringUnknown(), true, false},
		"project_id not known yet":               {types.StringUnknown(), types.StringValue("destination"), true, false},
		"project_id not known, no replication":   {types.StringUnknown(), types.StringValue("source"), false, false},
		"import_project_id is project_id, known": {types.StringValue("source"), types.StringValue("source"), true, true},
	} {
		t.Run(name, func(t *testing.T) {
			config := tfsdk.Config{Schema: s, Raw: secretImportTestPlan(t, s, secretImportTestModel(tc.projectID, tc.importProjectID, tc.isReplication)).Raw}
			resp := resource.ValidateConfigResponse{}
			(&projectSecretImportResource{}).ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: config}, &resp)

			if got := resp.Diagnostics.HasError(); got != tc.wantError {
				t.Errorf("ValidateConfig() error = %v, want %v: %v", got, tc.wantError, resp.Diagnostics)
			}
		})
	}
}
