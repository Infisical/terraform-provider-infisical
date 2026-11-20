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

// Fake backend for the secret import routes. Behavior and messages follow createImport, getImports
// and getImportById in Infisical core (secret-import-service.ts), and the response schemas in
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

func importSecretImport(t *testing.T, r *projectSecretImportResource, s schema.Schema, id string) projectSecretImportResourceModel {
	t.Helper()
	state := tfsdk.State{Schema: s}
	if diags := state.Set(context.Background(), &projectSecretImportResourceModel{}); diags.HasError() {
		t.Fatal(diags)
	}
	resp := resource.ImportStateResponse{State: state}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: id}, &resp)
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

	createResp := resource.CreateResponse{State: tfsdk.State{Schema: s}}
	r.Create(ctx, resource.CreateRequest{Plan: crossProjectImportPlan(t, s)}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatal(createResp.Diagnostics)
	}
	created := secretImportModel(t, createResp.State)
	if got := backend.imports[created.ID.ValueString()].SourceProjectID; got != "source" {
		t.Fatalf("expected the backend to import from project source, got %q", got)
	}

	readResp := resource.ReadResponse{State: createResp.State}
	r.Read(ctx, resource.ReadRequest{State: createResp.State}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatal(readResp.Diagnostics)
	}
	if got := secretImportModel(t, readResp.State).ImportProjectID; got != types.StringValue("source") {
		t.Errorf("expected import_project_id to survive a refresh, got %v", got)
	}

	imported := importSecretImport(t, r, s, created.ID.ValueString())
	created.ImportProjectID = types.StringValue("source")
	if imported != created {
		t.Errorf("import state = %+v, want %+v", imported, created)
	}

	backend.imports["same-project"] = fakeSecretImport{
		ProjectID: "destination", Environment: "prod", SecretPath: "/app",
		SourceProjectID: "destination", ImportEnv: "dev", ImportPath: "/common",
	}
	if got := importSecretImport(t, r, s, "same-project").ImportProjectID; !got.IsNull() {
		t.Errorf("expected a same-project import to leave import_project_id null, got %v", got)
	}

	for name, id := range map[string]string{"deleted import": "import-gone", "empty ID": ""} {
		t.Run(name, func(t *testing.T) {
			state := tfsdk.State{Schema: s, Raw: readResp.State.Raw.Copy()}
			if diags := state.SetAttribute(ctx, path.Root("id"), id); diags.HasError() {
				t.Fatal(diags)
			}
			resp := resource.ReadResponse{State: state}
			r.Read(ctx, resource.ReadRequest{State: state}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			if !resp.State.Raw.IsNull() {
				t.Error("expected the import to be removed from state")
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
