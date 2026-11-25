package resource

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type fakeSecretImport struct {
	ProjectID, Environment, SecretPath     string
	SourceProjectID, ImportEnv, ImportPath string
}

// Fake of the secret import routes, following createImport, getImports, getImportById and deleteImport in
// backend/src/services/secret-import/secret-import-service.ts.
type fakeSecretImportBackend struct {
	t                   *testing.T
	crossProjectSharing bool
	noProjectGrant      bool
	imports             map[string]fakeSecretImport
	lastCreatePath      string
}

func (b *fakeSecretImportBackend) writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		b.t.Error(err)
	}
}

func (b *fakeSecretImportBackend) handler() http.Handler {
	mux := http.NewServeMux()

	createImport := func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ProjectID   string `json:"projectId"`
			WorkspaceID string `json:"workspaceId"`
			Environment string `json:"environment"`
			Path        string `json:"path"`
			Import      struct {
				SourceProjectID string `json:"sourceProjectId"`
				Environment     string `json:"environment"`
				Path            string `json:"path"`
			} `json:"import"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			b.writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"statusCode": http.StatusUnprocessableEntity, "message": err.Error()})
			return
		}
		b.lastCreatePath = r.URL.Path
		projectID, sourceProjectID := body.ProjectID, body.Import.SourceProjectID
		if r.URL.Path == "/api/v1/secret-imports" {
			// v1 names the project workspaceId and drops import.sourceProjectId.
			projectID, sourceProjectID = body.WorkspaceID, ""
		}
		sourceProjectID = cmp.Or(sourceProjectID, projectID)
		if sourceProjectID != projectID && !b.crossProjectSharing {
			b.writeJSON(w, http.StatusForbidden, map[string]any{"statusCode": http.StatusForbidden, "message": "Cross-project secret sharing is not enabled for this organization"})
			return
		}
		if sourceProjectID != projectID && b.noProjectGrant {
			b.writeJSON(w, http.StatusForbidden, map[string]any{"statusCode": http.StatusForbidden, "message": "No project grant found allowing this cross-project import"})
			return
		}
		id := fmt.Sprintf("import-%d", len(b.imports)+1)
		b.imports[id] = fakeSecretImport{
			ProjectID: projectID, Environment: body.Environment, SecretPath: body.Path,
			SourceProjectID: sourceProjectID, ImportEnv: body.Import.Environment, ImportPath: body.Import.Path,
		}
		b.writeJSON(w, http.StatusOK, map[string]any{"secretImport": map[string]any{"id": id}})
	}
	mux.HandleFunc("POST /api/v1/secret-imports", createImport)
	mux.HandleFunc("POST /api/v2/secret-imports", createImport)

	mux.HandleFunc("GET /api/v2/secret-imports", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		list := []map[string]any{}
		for id, imp := range b.imports {
			if imp.ProjectID != q.Get("projectId") || imp.Environment != q.Get("environment") || imp.SecretPath != q.Get("path") {
				continue
			}
			if imp.SourceProjectID != imp.ProjectID && !b.crossProjectSharing {
				continue
			}
			list = append(list, map[string]any{"id": id, "importEnv": map[string]string{"slug": imp.ImportEnv, "projectId": imp.SourceProjectID}})
		}
		b.writeJSON(w, http.StatusOK, map[string]any{"secretImports": list})
	})

	mux.HandleFunc("GET /api/v1/secret-imports/{id}", func(w http.ResponseWriter, r *http.Request) {
		imp, ok := b.imports[r.PathValue("id")]
		if !ok {
			b.writeJSON(w, http.StatusNotFound, map[string]any{"statusCode": http.StatusNotFound, "message": "Secret import not found"})
			return
		}
		b.writeJSON(w, http.StatusOK, map[string]any{"secretImport": map[string]any{
			"id": r.PathValue("id"), "projectId": imp.ProjectID, "secretPath": imp.SecretPath, "importPath": imp.ImportPath,
			"environment": map[string]string{"slug": imp.Environment},
			"importEnv":   map[string]string{"slug": imp.ImportEnv},
		}})
	})

	// deleteImport finds the import in the destination folder and ignores the cross-project setting.
	mux.HandleFunc("DELETE /api/v1/secret-imports/{id}", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ProjectID   string `json:"workspaceId"`
			Environment string `json:"environment"`
			Path        string `json:"path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			b.writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"statusCode": http.StatusUnprocessableEntity, "message": err.Error()})
			return
		}
		id := r.PathValue("id")
		if imp, ok := b.imports[id]; !ok || imp.ProjectID != body.ProjectID || imp.Environment != body.Environment || imp.SecretPath != body.Path {
			b.writeJSON(w, http.StatusNotFound, map[string]any{"statusCode": http.StatusNotFound, "message": "Secret import not found"})
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

func secretImportTestPlan(t *testing.T, s schema.Schema, projectID, importProjectID types.String, isReplication bool) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: s}
	if diags := plan.Set(context.Background(), &projectSecretImportResourceModel{
		ID:              types.StringUnknown(),
		ProjectID:       projectID,
		EnvironmentSlug: types.StringValue("prod"),
		SecretPath:      types.StringValue("/app"),
		ImportPath:      types.StringValue("/shared"),
		ImportEnvSlug:   types.StringValue("dev"),
		ImportProjectID: importProjectID,
		IsReplication:   types.BoolValue(isReplication),
	}); diags.HasError() {
		t.Fatal(diags)
	}
	return plan
}

func createSecretImport(t *testing.T, r *projectSecretImportResource, s schema.Schema, importProjectID types.String) tfsdk.State {
	t.Helper()
	resp := resource.CreateResponse{State: tfsdk.State{Schema: s}}
	plan := secretImportTestPlan(t, s, types.StringValue("destination"), importProjectID, false)
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, &resp)
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

func importSecretImport(t *testing.T, r *projectSecretImportResource, s schema.Schema, id string) resource.ImportStateResponse {
	t.Helper()
	resp := resource.ImportStateResponse{State: tfsdk.State{Schema: s}}
	if diags := resp.State.Set(context.Background(), &projectSecretImportResourceModel{}); diags.HasError() {
		t.Fatal(diags)
	}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: id}, &resp)
	return resp
}

func secretImportTestModel(t *testing.T, state tfsdk.State) projectSecretImportResourceModel {
	t.Helper()
	var model projectSecretImportResourceModel
	if diags := state.Get(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return model
}

func TestSecretImportCrossProjectLifecycle(t *testing.T) {
	backend := &fakeSecretImportBackend{t: t, crossProjectSharing: true, imports: map[string]fakeSecretImport{}}
	r := newSecretImportTestResource(t, backend)
	s := secretImportTestSchema(t)

	state := createSecretImport(t, r, s, types.StringValue("source"))
	created := secretImportTestModel(t, state)
	id := created.ID.ValueString()
	want := fakeSecretImport{
		ProjectID: "destination", Environment: "prod", SecretPath: "/app",
		SourceProjectID: "source", ImportEnv: "dev", ImportPath: "/shared",
	}
	if got := backend.imports[id]; got != want {
		t.Fatalf("backend import = %+v, want %+v", got, want)
	}

	state = readSecretImport(t, r, state)
	if got := secretImportTestModel(t, state); got != created {
		t.Errorf("state after refresh = %+v, want %+v", got, created)
	}

	imported := importSecretImport(t, r, s, id)
	if imported.Diagnostics.HasError() {
		t.Fatal(imported.Diagnostics)
	}
	if got := secretImportTestModel(t, imported.State); got != created {
		t.Errorf("state after import = %+v, want %+v", got, created)
	}

	want.SecretPath = "/renamed"
	backend.imports[id] = want
	state = readSecretImport(t, r, state)
	if got := secretImportTestModel(t, state).SecretPath.ValueString(); got != "/renamed" {
		t.Errorf("folder_path after the destination folder was renamed = %q, want /renamed", got)
	}

	deleteResp := resource.DeleteResponse{State: state}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, &deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatal(deleteResp.Diagnostics)
	}
	if _, ok := backend.imports[id]; ok {
		t.Fatal("expected Delete to remove the import")
	}

	if !readSecretImport(t, r, state).Raw.IsNull() {
		t.Error("expected Read to remove the deleted import from state")
	}
}

// Existing configs have no import_project_id, so Read must still find these imports by ID.
func TestSecretImportSameProjectLifecycle(t *testing.T) {
	backend := &fakeSecretImportBackend{t: t, imports: map[string]fakeSecretImport{}}
	r := newSecretImportTestResource(t, backend)
	s := secretImportTestSchema(t)

	state := createSecretImport(t, r, s, types.StringNull())
	created := secretImportTestModel(t, state)
	// Same-project creates stay on v1, which older servers also have.
	want := fakeSecretImport{
		ProjectID: "destination", Environment: "prod", SecretPath: "/app",
		SourceProjectID: "destination", ImportEnv: "dev", ImportPath: "/shared",
	}
	if got := backend.imports[created.ID.ValueString()]; backend.lastCreatePath != "/api/v1/secret-imports" || got != want {
		t.Fatalf("create via %s saved %+v, want /api/v1/secret-imports and %+v", backend.lastCreatePath, got, want)
	}

	refreshed := readSecretImport(t, r, state)
	if refreshed.Raw.IsNull() {
		t.Fatal("expected Read to keep the import in state")
	}
	if got := secretImportTestModel(t, refreshed); got != created {
		t.Errorf("state after refresh = %+v, want %+v", got, created)
	}

	imported := importSecretImport(t, r, s, created.ID.ValueString())
	if imported.Diagnostics.HasError() {
		t.Fatal(imported.Diagnostics)
	}
	if got := secretImportTestModel(t, imported.State); got != created {
		t.Errorf("state after import = %+v, want %+v", got, created)
	}

	deleteResp := resource.DeleteResponse{State: refreshed}
	r.Delete(context.Background(), resource.DeleteRequest{State: refreshed}, &deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatal(deleteResp.Diagnostics)
	}
	if !readSecretImport(t, r, refreshed).Raw.IsNull() {
		t.Error("expected Read to remove the deleted import from state")
	}
}

// The v2 list hides cross-project imports while cross-project secret sharing is turned off, but they still exist.
func TestSecretImportWithCrossProjectSharingTurnedOff(t *testing.T) {
	backend := &fakeSecretImportBackend{t: t, crossProjectSharing: true, imports: map[string]fakeSecretImport{}}
	r := newSecretImportTestResource(t, backend)
	s := secretImportTestSchema(t)

	state := createSecretImport(t, r, s, types.StringValue("source"))
	created := secretImportTestModel(t, state)
	backend.crossProjectSharing = false

	refreshed := readSecretImport(t, r, state)
	if refreshed.Raw.IsNull() {
		t.Fatal("expected Read to keep the import in state")
	}
	if got := secretImportTestModel(t, refreshed); got != created {
		t.Errorf("state after refresh = %+v, want %+v", got, created)
	}

	// Importing without the source project would save a null import_project_id, and the next plan would replace the import.
	imported := importSecretImport(t, r, s, created.ID.ValueString())
	if errs := imported.Diagnostics.Errors(); len(errs) != 1 || !strings.Contains(errs[0].Detail(), "Turn it on under Secrets Management > Product Settings") {
		t.Errorf("expected import to fail and ask to turn on cross-project secret sharing, got %v", imported.Diagnostics)
	}
}

func TestSecretImportCreateExplainsCrossProjectSetup(t *testing.T) {
	s := secretImportTestSchema(t)

	for name, tc := range map[string]struct {
		crossProjectSharing, noProjectGrant bool
		wantMessage, wantHint               string
	}{
		"sharing turned off": {
			wantMessage: "Cross-project secret sharing is not enabled for this organization",
			wantHint:    "An organization admin must turn on cross-project secret sharing under Secrets Management > Product Settings.",
		},
		"no project grant": {
			crossProjectSharing: true, noProjectGrant: true,
			wantMessage: "No project grant found allowing this cross-project import",
			wantHint:    `In project "source", add a project grant under Project Settings > Policies that shares folder "/shared" in environment "dev" with project "destination".`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			backend := &fakeSecretImportBackend{t: t, crossProjectSharing: tc.crossProjectSharing, noProjectGrant: tc.noProjectGrant, imports: map[string]fakeSecretImport{}}
			r := newSecretImportTestResource(t, backend)

			resp := resource.CreateResponse{State: tfsdk.State{Schema: s}}
			plan := secretImportTestPlan(t, s, types.StringValue("destination"), types.StringValue("source"), false)
			r.Create(context.Background(), resource.CreateRequest{Plan: plan}, &resp)

			errs := resp.Diagnostics.Errors()
			if len(errs) != 1 {
				t.Fatalf("expected one error, got %v", resp.Diagnostics)
			}
			if withPath, ok := errs[0].(diag.DiagnosticWithPath); !ok || !withPath.Path().Equal(path.Root("import_project_id")) {
				t.Errorf("expected the error on import_project_id, got %v", errs[0])
			}
			for _, want := range []string{tc.wantMessage, tc.wantHint} {
				if !strings.Contains(errs[0].Detail(), want) {
					t.Errorf("expected the detail to contain %q, got:\n%s", want, errs[0].Detail())
				}
			}
		})
	}
}

func TestSecretImportValidateConfigRejectsCrossProjectReplication(t *testing.T) {
	s := secretImportTestSchema(t)

	for name, tc := range map[string]struct {
		projectID, importProjectID types.String
		wantError                  bool
	}{
		"cross-project":                   {types.StringValue("destination"), types.StringValue("source"), true},
		"same project":                    {types.StringValue("destination"), types.StringNull(), false},
		"import_project_id is project_id": {types.StringValue("destination"), types.StringValue("destination"), false},
		"import_project_id not known yet": {types.StringValue("destination"), types.StringUnknown(), false},
		"project_id not known yet":        {types.StringUnknown(), types.StringValue("destination"), false},
	} {
		t.Run(name, func(t *testing.T) {
			config := tfsdk.Config{Schema: s, Raw: secretImportTestPlan(t, s, tc.projectID, tc.importProjectID, true).Raw}
			resp := resource.ValidateConfigResponse{}
			(&projectSecretImportResource{}).ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: config}, &resp)

			if got := resp.Diagnostics.HasError(); got != tc.wantError {
				t.Errorf("ValidateConfig() error = %v, want %v: %v", got, tc.wantError, resp.Diagnostics)
			}
		})
	}
}
