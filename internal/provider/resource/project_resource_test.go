package resource

import (
	"context"
	"encoding/json"
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

const (
	testProjectID          = "b2c0d6a4-7e1f-4c3a-9d5b-8f2e1a0c4b6d"
	testProjectDescription = "Owner: platform-team@example.com"
)

// Fake of GET /api/v1/workspace/:workspaceId (backend/src/server/routes/v1/deprecated-project-router.ts)
// and PATCH /api/v1/projects/:projectId (backend/src/server/routes/v1/project-router.ts): the PATCH
// stores the trimmed description and the GET returns it.
type fakeProjectBackend struct {
	t           *testing.T
	exists      bool
	description string
}

func (b *fakeProjectBackend) writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		b.t.Error(err)
	}
}

func (b *fakeProjectBackend) project() map[string]any {
	return map[string]any{
		"id":                     testProjectID,
		"name":                   "My Project",
		"slug":                   "my-project",
		"type":                   "secret-manager",
		"description":            b.description,
		"kmsSecretManagerKeyId":  "kms-key-id",
		"auditLogsRetentionDays": 30,
		"updatedAt":              "2026-11-25T00:00:00.000Z",
	}
}

func (b *fakeProjectBackend) get(w http.ResponseWriter, _ *http.Request) {
	if !b.exists {
		b.writeJSON(w, http.StatusNotFound, map[string]any{"statusCode": http.StatusNotFound, "message": "Project not found"})
		return
	}
	b.writeJSON(w, http.StatusOK, map[string]any{"workspace": b.project()})
}

func (b *fakeProjectBackend) update(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Description *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		b.t.Error(err)
		return
	}
	if body.Description != nil {
		b.description = strings.TrimSpace(*body.Description)
	}
	b.writeJSON(w, http.StatusOK, map[string]any{"project": b.project()})
}

func newTestProjectResource(t *testing.T, backend *fakeProjectBackend) *projectResource {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/workspace/{id}", backend.get)
	mux.HandleFunc("PATCH /api/v1/projects/{id}", backend.update)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return &projectResource{client: &infisical.Client{Config: infisical.Config{
		HttpClient:            resty.New().SetBaseURL(server.URL),
		IsMachineIdentityAuth: true,
	}}}
}

func projectTestSchema(t *testing.T) schema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	NewProjectResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

// projectTestModel returns a project as stored in state, with the fields under test left null.
func projectTestModel() projectResourceModel {
	return projectResourceModel{
		ID:                    types.StringValue(testProjectID),
		Name:                  types.StringValue("My Project"),
		Slug:                  types.StringValue("my-project"),
		Type:                  types.StringValue("secret-manager"),
		KmsSecretManagerKeyId: types.StringValue("kms-key-id"),
		HasDeleteProtection:   types.BoolValue(false),
		AuditLogRetentionDays: types.Int64Value(30),
	}
}

func projectTestState(t *testing.T, s schema.Schema, model projectResourceModel) tfsdk.State {
	t.Helper()
	state := tfsdk.State{Schema: s}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return state
}

func projectStateModel(t *testing.T, state tfsdk.State) projectResourceModel {
	t.Helper()
	var model projectResourceModel
	if diags := state.Get(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return model
}

func TestProjectDescriptionWOSchema(t *testing.T) {
	attribute := projectTestSchema(t).Attributes["description_wo"]
	if !attribute.IsWriteOnly() || !attribute.IsSensitive() {
		t.Error("expected description_wo to be write-only and sensitive")
	}
}

func TestProjectReadDescription(t *testing.T) {
	s := projectTestSchema(t)

	for name, tc := range map[string]struct {
		descriptionWOVersion types.Int64
		want                 types.String
	}{
		"description":    {descriptionWOVersion: types.Int64Null(), want: types.StringValue(testProjectDescription)},
		"description_wo": {descriptionWOVersion: types.Int64Value(1), want: types.StringNull()},
	} {
		t.Run(name, func(t *testing.T) {
			r := newTestProjectResource(t, &fakeProjectBackend{t: t, exists: true, description: testProjectDescription})

			model := projectTestModel()
			model.DescriptionWOVersion = tc.descriptionWOVersion
			state := projectTestState(t, s, model)
			resp := resource.ReadResponse{State: state}
			r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)

			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			if got := projectStateModel(t, resp.State).Description; !got.Equal(tc.want) {
				t.Errorf("description in state: got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestProjectReadRemovesMissingProject(t *testing.T) {
	s := projectTestSchema(t)

	for name, id := range map[string]string{
		"deleted in Infisical": testProjectID,
		"empty id":             "",
	} {
		t.Run(name, func(t *testing.T) {
			r := newTestProjectResource(t, &fakeProjectBackend{t: t})

			model := projectTestModel()
			model.ID = types.StringValue(id)
			state := projectTestState(t, s, model)
			resp := resource.ReadResponse{State: state}
			r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)

			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			if !resp.State.Raw.IsNull() {
				t.Error("expected the project to be removed from state")
			}
		})
	}
}

// Moving from description to description_wo sends the write-only value to Infisical and
// removes the description from state.
func TestProjectUpdateToDescriptionWO(t *testing.T) {
	s := projectTestSchema(t)
	backend := &fakeProjectBackend{t: t, exists: true, description: "old description"}
	r := newTestProjectResource(t, backend)

	prior := projectTestModel()
	prior.Description = types.StringValue("old description")

	planned := projectTestModel()
	planned.DescriptionWOVersion = types.Int64Value(1)

	configured := planned
	configured.DescriptionWO = types.StringValue(testProjectDescription)

	state := projectTestState(t, s, prior)
	resp := resource.UpdateResponse{State: tfsdk.State{Schema: s}}
	r.Update(context.Background(), resource.UpdateRequest{
		Config: tfsdk.Config{Schema: s, Raw: projectTestState(t, s, configured).Raw},
		Plan:   tfsdk.Plan{Schema: s, Raw: projectTestState(t, s, planned).Raw},
		State:  state,
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if backend.description != testProjectDescription {
		t.Errorf("description sent to Infisical: got %q, want %q", backend.description, testProjectDescription)
	}
	model := projectStateModel(t, resp.State)
	if !model.Description.IsNull() || !model.DescriptionWO.IsNull() {
		t.Errorf("expected the description to stay out of state, got description=%v description_wo=%v", model.Description, model.DescriptionWO)
	}
	if model.DescriptionWOVersion.ValueInt64() != 1 {
		t.Errorf("description_wo_version: got %v, want 1", model.DescriptionWOVersion)
	}
}
