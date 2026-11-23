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
	IsReplication                          bool
}

// Fake backend for the secret import routes. Behavior and messages follow createImport, deleteImport,
// getImports and getImportById in Infisical core (secret-import-service.ts), and the response schemas in
// secret-import-router.ts (v2) and deprecated-secret-import-router.ts (v1).
type fakeSecretImportBackend struct {
	t                   *testing.T
	crossProjectSharing bool
	// "sourceProject envSlug folderPath targetProject".
	grants  map[string]bool
	imports map[string]fakeSecretImport
}

func newFakeSecretImportBackend(t *testing.T) *fakeSecretImportBackend {
	return &fakeSecretImportBackend{
		t:                   t,
		crossProjectSharing: true,
		grants:              map[string]bool{"source dev /shared destination": true},
		imports:             map[string]fakeSecretImport{},
	}
}

func (b *fakeSecretImportBackend) writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		b.t.Error(err)
	}
}

func (b *fakeSecretImportBackend) apiError(w http.ResponseWriter, status int, name, message string) {
	b.writeJSON(w, status, map[string]any{"reqId": "req-1", "statusCode": status, "message": message, "error": name})
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
			b.apiError(w, http.StatusUnprocessableEntity, "ValidationFailure", err.Error())
			return
		}

		source := body.Import.SourceProjectID
		if source == "" {
			source = body.ProjectID
		}
		if source != body.ProjectID {
			if !b.crossProjectSharing {
				b.apiError(w, http.StatusForbidden, "ForbiddenError", "Cross-project secret sharing is not enabled for this organization")
				return
			}
			if !b.grants[strings.Join([]string{source, body.Import.Environment, body.Import.Path, body.ProjectID}, " ")] {
				b.apiError(w, http.StatusForbidden, "ForbiddenError", "No project grant found allowing this cross-project import")
				return
			}
			if body.IsReplication {
				b.apiError(w, http.StatusBadRequest, "BadRequest", "Replication is not supported for cross-project imports")
				return
			}
		}

		id := fmt.Sprintf("import-%d", len(b.imports)+1)
		b.imports[id] = fakeSecretImport{
			ProjectID: body.ProjectID, Environment: body.Environment, SecretPath: body.Path,
			SourceProjectID: source, ImportEnv: body.Import.Environment, ImportPath: body.Import.Path,
			IsReplication: body.IsReplication,
		}
		// The response schema keeps only id, name and slug of importEnv.
		b.writeJSON(w, http.StatusOK, map[string]any{"message": "Successfully created secret import", "secretImport": map[string]any{
			"id": id, "importPath": body.Import.Path, "isReplication": body.IsReplication,
			"importEnv": map[string]string{"id": "env-" + body.Import.Environment, "name": body.Import.Environment, "slug": body.Import.Environment},
		}})
	})

	mux.HandleFunc("GET /api/v2/secret-imports", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		list := []map[string]any{}
		for id, imp := range b.imports {
			if imp.ProjectID != q.Get("projectId") || imp.Environment != q.Get("environment") || imp.SecretPath != q.Get("path") {
				continue
			}
			// getImports drops cross-project imports while the organization toggle is off.
			if imp.SourceProjectID != imp.ProjectID && !b.crossProjectSharing {
				continue
			}
			list = append(list, map[string]any{
				"id": id, "importPath": imp.ImportPath, "isReplication": imp.IsReplication,
				"importEnv": map[string]string{"id": "env-" + imp.ImportEnv, "name": imp.ImportEnv, "slug": imp.ImportEnv, "projectId": imp.SourceProjectID},
			})
		}
		b.writeJSON(w, http.StatusOK, map[string]any{"message": "Successfully fetched secret imports", "secretImports": list})
	})

	// deleteImport finds the destination folder from workspaceId, environment and path, and deletes the import
	// in it. It checks neither the source project nor the organization's cross-project setting.
	mux.HandleFunc("DELETE /api/v1/secret-imports/{id}", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ProjectID   string `json:"workspaceId"`
			Environment string `json:"environment"`
			Path        string `json:"path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			b.apiError(w, http.StatusUnprocessableEntity, "ValidationFailure", err.Error())
			return
		}

		id := r.PathValue("id")
		imp, ok := b.imports[id]
		if !ok || imp.ProjectID != body.ProjectID || imp.Environment != body.Environment || imp.SecretPath != body.Path {
			b.apiError(w, http.StatusNotFound, "NotFound", fmt.Sprintf("Secret import with folder ID '%s' not found", id))
			return
		}
		delete(b.imports, id)
		b.writeJSON(w, http.StatusOK, map[string]any{"message": "Successfully deleted secret import", "secretImport": map[string]any{
			"id": id, "importPath": imp.ImportPath, "isReplication": imp.IsReplication,
			"importEnv": map[string]string{"id": "env-" + imp.ImportEnv, "name": imp.ImportEnv, "slug": imp.ImportEnv},
		}})
	})

	mux.HandleFunc("GET /api/v1/secret-imports/{id}", func(w http.ResponseWriter, r *http.Request) {
		imp, ok := b.imports[r.PathValue("id")]
		if !ok {
			b.apiError(w, http.StatusNotFound, "NotFound", fmt.Sprintf("Secret import with ID '%s' not found", r.PathValue("id")))
			return
		}
		b.writeJSON(w, http.StatusOK, map[string]any{"secretImport": map[string]any{
			"id": r.PathValue("id"), "importPath": imp.ImportPath, "isReplication": imp.IsReplication,
			"projectId": imp.ProjectID, "secretPath": imp.SecretPath,
			"environment": map[string]string{"id": "env-" + imp.Environment, "name": imp.Environment, "slug": imp.Environment},
			"importEnv":   map[string]string{"id": "env-" + imp.ImportEnv, "name": imp.ImportEnv, "slug": imp.ImportEnv},
		}})
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

func crossProjectImportPlan(t *testing.T, s schema.Schema) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: s}
	if diags := plan.Set(context.Background(), &projectSecretImportResourceModel{
		ID:              types.StringUnknown(),
		ProjectID:       types.StringValue("destination"),
		EnvironmentSlug: types.StringValue("prod"),
		SecretPath:      types.StringValue("/app"),
		ImportPath:      types.StringValue("/shared"),
		ImportEnvSlug:   types.StringValue("dev"),
		ImportProjectID: types.StringValue("source"),
		IsReplication:   types.BoolValue(false),
	}); diags.HasError() {
		t.Fatal(diags)
	}
	return plan
}

func secretImportModel(t *testing.T, state tfsdk.State) projectSecretImportResourceModel {
	t.Helper()
	var model projectSecretImportResourceModel
	if diags := state.Get(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return model
}

func createCrossProjectImport(t *testing.T, r *projectSecretImportResource, s schema.Schema) tfsdk.State {
	t.Helper()
	resp := resource.CreateResponse{State: tfsdk.State{Schema: s}}
	r.Create(context.Background(), resource.CreateRequest{Plan: crossProjectImportPlan(t, s)}, &resp)
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

func runImportState(t *testing.T, r *projectSecretImportResource, s schema.Schema, id string) resource.ImportStateResponse {
	t.Helper()
	state := tfsdk.State{Schema: s}
	if diags := state.Set(context.Background(), &projectSecretImportResourceModel{}); diags.HasError() {
		t.Fatal(diags)
	}
	resp := resource.ImportStateResponse{State: state}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: id}, &resp)
	return resp
}

func importSecretImport(t *testing.T, r *projectSecretImportResource, s schema.Schema, id string) projectSecretImportResourceModel {
	t.Helper()
	resp := runImportState(t, r, s, id)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	return secretImportModel(t, resp.State)
}

func TestSecretImportCrossProjectLifecycle(t *testing.T) {
	ctx := context.Background()
	s := secretImportTestSchema(t)
	backend := newFakeSecretImportBackend(t)
	r := newSecretImportTestResource(t, backend)

	state := createCrossProjectImport(t, r, s)
	created := secretImportModel(t, state)
	id := created.ID.ValueString()
	if got := backend.imports[id].SourceProjectID; got != "source" {
		t.Fatalf("expected the backend to import from project source, got %q", got)
	}

	state = readSecretImport(t, r, state)
	if got := secretImportModel(t, state); got != created {
		t.Errorf("state after refresh = %+v, want %+v", got, created)
	}

	if imported := importSecretImport(t, r, s, id); imported != created {
		t.Errorf("import state = %+v, want %+v", imported, created)
	}

	backend.imports["same-project"] = fakeSecretImport{
		ProjectID: "destination", Environment: "prod", SecretPath: "/app",
		SourceProjectID: "destination", ImportEnv: "dev", ImportPath: "/common",
	}
	if got := importSecretImport(t, r, s, "same-project").ImportProjectID; !got.IsNull() {
		t.Errorf("expected a same-project import to leave import_project_id null, got %v", got)
	}

	deleteResp := resource.DeleteResponse{State: state}
	r.Delete(ctx, resource.DeleteRequest{State: state}, &deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatal(deleteResp.Diagnostics)
	}
	if _, ok := backend.imports[id]; ok {
		t.Fatal("expected Delete to remove the import from the backend")
	}

	if !readSecretImport(t, r, state).Raw.IsNull() {
		t.Error("expected Read to remove the deleted import from state")
	}

	emptyID := tfsdk.State{Schema: s, Raw: state.Raw.Copy()}
	if diags := emptyID.SetAttribute(ctx, path.Root("id"), ""); diags.HasError() {
		t.Fatal(diags)
	}
	if !readSecretImport(t, r, emptyID).Raw.IsNull() {
		t.Error("expected Read to remove an import with an empty ID from state")
	}
}

// With cross-project secret sharing turned off, the v2 list leaves out cross-project imports, but they still
// exist and the get by ID route still returns them.
func TestSecretImportCrossProjectSharingTurnedOff(t *testing.T) {
	s := secretImportTestSchema(t)
	backend := newFakeSecretImportBackend(t)
	r := newSecretImportTestResource(t, backend)

	state := createCrossProjectImport(t, r, s)
	created := secretImportModel(t, state)
	backend.crossProjectSharing = false

	refreshed := readSecretImport(t, r, state)
	if refreshed.Raw.IsNull() {
		t.Fatal("expected Read to keep the import in state")
	}
	if got := secretImportModel(t, refreshed); got != created {
		t.Errorf("state after refresh = %+v, want %+v", got, created)
	}

	resp := runImportState(t, r, s, created.ID.ValueString())
	if errs := resp.Diagnostics.Errors(); len(errs) != 1 || !strings.Contains(errs[0].Detail(), "cross-project secret sharing is turned on") {
		t.Errorf("expected import to fail and ask to turn on cross-project secret sharing, got %v", resp.Diagnostics)
	}
}

func TestSecretImportValidateConfigRejectsCrossProjectReplication(t *testing.T) {
	ctx := context.Background()
	s := secretImportTestSchema(t)

	for name, tc := range map[string]struct {
		importProjectID types.String
		isReplication   bool
		wantError       bool
	}{
		"cross-project replication":         {importProjectID: types.StringValue("source"), isReplication: true, wantError: true},
		"cross-project without replication": {importProjectID: types.StringValue("source")},
		"same-project replication":          {importProjectID: types.StringNull(), isReplication: true},
		"source project is project_id":      {importProjectID: types.StringValue("destination"), isReplication: true},
		"source project not known yet":      {importProjectID: types.StringUnknown(), isReplication: true},
	} {
		t.Run(name, func(t *testing.T) {
			config := crossProjectImportPlan(t, s)
			config.Raw = config.Raw.Copy()
			diags := config.SetAttribute(ctx, path.Root("import_project_id"), tc.importProjectID)
			diags.Append(config.SetAttribute(ctx, path.Root("is_replication"), tc.isReplication)...)
			if diags.HasError() {
				t.Fatal(diags)
			}

			resp := resource.ValidateConfigResponse{}
			r := &projectSecretImportResource{}
			r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: s, Raw: config.Raw}}, &resp)

			errs := resp.Diagnostics.Errors()
			if got := len(errs) > 0; got != tc.wantError {
				t.Fatalf("ValidateConfig() error = %v, want %v: %v", got, tc.wantError, resp.Diagnostics)
			}
			if !tc.wantError {
				return
			}
			if withPath, ok := errs[0].(diag.DiagnosticWithPath); !ok || !withPath.Path().Equal(path.Root("is_replication")) {
				t.Errorf("expected the error on is_replication, got %v", errs[0])
			}
		})
	}
}

func TestSecretImportCrossProjectCreateErrors(t *testing.T) {
	s := secretImportTestSchema(t)

	for name, tc := range map[string]struct {
		sharing  bool
		grants   map[string]bool
		wantHint string
	}{
		"sharing turned off": {sharing: false, grants: map[string]bool{"source dev /shared destination": true}, wantHint: `turn on "Cross-project secret sharing"`},
		"no project grant":   {sharing: true, grants: map[string]bool{}, wantHint: `Project "source" has no project grant`},
	} {
		t.Run(name, func(t *testing.T) {
			backend := newFakeSecretImportBackend(t)
			backend.crossProjectSharing = tc.sharing
			backend.grants = tc.grants
			r := newSecretImportTestResource(t, backend)

			resp := resource.CreateResponse{State: tfsdk.State{Schema: s}}
			r.Create(context.Background(), resource.CreateRequest{Plan: crossProjectImportPlan(t, s)}, &resp)

			errs := resp.Diagnostics.Errors()
			if len(errs) != 1 {
				t.Fatalf("expected one error, got %v", resp.Diagnostics)
			}
			withPath, ok := errs[0].(diag.DiagnosticWithPath)
			if !ok || !withPath.Path().Equal(path.Root("import_project_id")) {
				t.Errorf("expected the error on import_project_id, got %v", errs[0])
			}
			for _, want := range []string{`project "source"`, `into project "destination"`, tc.wantHint, "status-code=403"} {
				if !strings.Contains(errs[0].Detail(), want) {
					t.Errorf("expected the detail to contain %q, got:\n%s", want, errs[0].Detail())
				}
			}
			if len(backend.imports) != 0 {
				t.Errorf("expected no import on the backend, got %v", backend.imports)
			}
		})
	}
}
