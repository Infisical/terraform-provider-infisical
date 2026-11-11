package resource

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	testSecretID    = "6f1c1d8e-0a52-4b8e-9a8d-1d2c3b4a5f60"
	testSecretValue = "plaintext"
	// INFISICAL_SECRET_VALUE_HIDDEN_MASK in backend/src/services/secret/secret-fns.ts.
	testHiddenSecretMask = "<hidden-by-infisical>"
)

// Fake of GET /api/v4/secrets/id/:secretId and GET /api/v3/secrets/raw/:secretName for a caller
// without Read Value, mirroring getSecretById and getSecretByName in
// backend/src/services/secret-v2-bridge/secret-v2-bridge-service.ts: viewSecretValue defaults to
// true and then needs Read Value (403), while false needs only Describe Secret and masks the value.
type fakeSecretBackend struct {
	t               *testing.T
	canReadValue    bool
	exists          bool
	viewSecretValue []string
}

func (b *fakeSecretBackend) writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		b.t.Error(err)
	}
}

func (b *fakeSecretBackend) secret(w http.ResponseWriter, r *http.Request) {
	viewSecretValue := r.URL.Query().Get("viewSecretValue")
	b.viewSecretValue = append(b.viewSecretValue, viewSecretValue)

	if !b.exists {
		b.writeJSON(w, http.StatusNotFound, map[string]any{"statusCode": http.StatusNotFound, "message": "Secret not found"})
		return
	}

	value, hidden := testSecretValue, false
	if viewSecretValue == "false" {
		value, hidden = testHiddenSecretMask, true
	} else if !b.canReadValue {
		b.writeJSON(w, http.StatusForbidden, map[string]any{"statusCode": http.StatusForbidden, "message": "You are not allowed to readValue on secrets"})
		return
	}

	b.writeJSON(w, http.StatusOK, map[string]any{"secret": map[string]any{
		"id":                testSecretID,
		"version":           1,
		"workspace":         "project-id",
		"type":              "shared",
		"environment":       "dev",
		"secretPath":        "/app",
		"secretKey":         "API_KEY",
		"secretValue":       value,
		"secretValueHidden": hidden,
		"updatedAt":         "2026-11-11T00:00:00.000Z",
	}})
}

func newTestSecretResource(t *testing.T, backend *fakeSecretBackend) *secretResource {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v4/secrets/id/{id}", backend.secret)
	mux.HandleFunc("GET /api/v3/secrets/raw/{name}", backend.secret)
	// GET /api/v4/secrets/id/ with no ID falls through to GET /api/v4/secrets/:secretName, which
	// refuses a request without a projectId.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		backend.viewSecretValue = append(backend.viewSecretValue, r.URL.Query().Get("viewSecretValue"))
		backend.writeJSON(w, http.StatusBadRequest, map[string]any{"statusCode": http.StatusBadRequest, "message": "You must provide  workspaceId"})
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return &secretResource{client: &infisical.Client{Config: infisical.Config{
		HttpClient:            resty.New().SetBaseURL(server.URL),
		IsMachineIdentityAuth: true,
	}}}
}

func secretTestSchema(t *testing.T) schema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	NewSecretResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func secretTestState(t *testing.T, s schema.Schema, id string, value types.String, valueWOVersion types.Int64) tfsdk.State {
	t.Helper()
	state := tfsdk.State{Schema: s}
	if diags := state.Set(context.Background(), &secretResourceModel{
		ID:             types.StringValue(id),
		Name:           types.StringValue("API_KEY"),
		FolderPath:     types.StringValue("/app"),
		EnvSlug:        types.StringValue("dev"),
		WorkspaceId:    types.StringValue("project-id"),
		Value:          value,
		ValueWO:        types.StringNull(),
		ValueWOVersion: valueWOVersion,
		LastUpdated:    types.StringNull(),
		Tags:           types.SetNull(types.StringType),
		Metadata:       types.MapNull(types.StringType),
	}); diags.HasError() {
		t.Fatal(diags)
	}
	return state
}

func secretTestModel(t *testing.T, state tfsdk.State) secretResourceModel {
	t.Helper()
	var model secretResourceModel
	if diags := state.Get(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return model
}

func readTestSecret(t *testing.T, r *secretResource, state tfsdk.State) resource.ReadResponse {
	t.Helper()
	resp := resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
	return resp
}

func TestSecretReadWriteOnlyWithoutReadValue(t *testing.T) {
	backend := &fakeSecretBackend{t: t, exists: true}
	r := newTestSecretResource(t, backend)

	resp := readTestSecret(t, r, secretTestState(t, secretTestSchema(t), testSecretID, types.StringNull(), types.Int64Value(1)))

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected a write-only refresh to work without Read Value, got: %v", resp.Diagnostics)
	}
	if len(backend.viewSecretValue) != 1 || backend.viewSecretValue[0] != "false" {
		t.Fatalf("expected one read with viewSecretValue=false, got %v", backend.viewSecretValue)
	}
	model := secretTestModel(t, resp.State)
	if !model.Value.IsNull() {
		t.Errorf("expected value to stay out of state, got %q", model.Value.ValueString())
	}
	if model.Name.ValueString() != "API_KEY" || model.FolderPath.ValueString() != "/app" || model.EnvSlug.ValueString() != "dev" {
		t.Errorf("unexpected state after refresh: %+v", model)
	}
}

func TestSecretReadRegularValueRefreshesValue(t *testing.T) {
	backend := &fakeSecretBackend{t: t, exists: true, canReadValue: true}
	r := newTestSecretResource(t, backend)

	resp := readTestSecret(t, r, secretTestState(t, secretTestSchema(t), testSecretID, types.StringValue("stale"), types.Int64Null()))

	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if len(backend.viewSecretValue) != 1 || backend.viewSecretValue[0] != "true" {
		t.Fatalf("expected one read with viewSecretValue=true, got %v", backend.viewSecretValue)
	}
	if got := secretTestModel(t, resp.State).Value.ValueString(); got != testSecretValue {
		t.Errorf("expected the refreshed value %q in state, got %q", testSecretValue, got)
	}
}

func TestSecretReadRemovesMissingSecret(t *testing.T) {
	s := secretTestSchema(t)

	for name, tc := range map[string]struct {
		id        string
		wantReads int
	}{
		"deleted in Infisical": {id: testSecretID, wantReads: 1},
		"empty id":             {id: "", wantReads: 0},
	} {
		t.Run(name, func(t *testing.T) {
			backend := &fakeSecretBackend{t: t, canReadValue: true}
			r := newTestSecretResource(t, backend)

			resp := readTestSecret(t, r, secretTestState(t, s, tc.id, types.StringValue(testSecretValue), types.Int64Null()))

			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			if !resp.State.Raw.IsNull() {
				t.Error("expected the secret to be removed from state")
			}
			if len(backend.viewSecretValue) != tc.wantReads {
				t.Errorf("expected %d reads, got %d", tc.wantReads, len(backend.viewSecretValue))
			}
		})
	}
}

func TestSecretImportReadsValueOnlyWhenNotWriteOnly(t *testing.T) {
	ctx := context.Background()
	s := secretTestSchema(t)

	for name, tc := range map[string]struct {
		importID      string
		canReadValue  bool
		wantView      string
		wantValue     types.String
		wantWOVersion types.Int64
	}{
		"write-only by id":   {importID: "write-only:" + testSecretID, wantView: "false", wantValue: types.StringNull(), wantWOVersion: types.Int64Value(1)},
		"write-only by name": {importID: "write-only:project-id:dev:/app:API_KEY", wantView: "false", wantValue: types.StringNull(), wantWOVersion: types.Int64Value(1)},
		"regular by id":      {importID: testSecretID, canReadValue: true, wantView: "true", wantValue: types.StringValue(testSecretValue), wantWOVersion: types.Int64Null()},
		"regular by name":    {importID: "project-id:dev:/app:API_KEY", canReadValue: true, wantView: "true", wantValue: types.StringValue(testSecretValue), wantWOVersion: types.Int64Null()},
	} {
		t.Run(name, func(t *testing.T) {
			backend := &fakeSecretBackend{t: t, exists: true, canReadValue: tc.canReadValue}
			r := newTestSecretResource(t, backend)

			importState := tfsdk.State{Schema: s}
			if diags := importState.Set(ctx, &secretResourceModel{
				Value:          types.StringNull(),
				ValueWO:        types.StringNull(),
				ValueWOVersion: types.Int64Null(),
				Tags:           types.SetNull(types.StringType),
				Metadata:       types.MapNull(types.StringType),
			}); diags.HasError() {
				t.Fatal(diags)
			}
			resp := resource.ImportStateResponse{State: importState}
			r.ImportState(ctx, resource.ImportStateRequest{ID: tc.importID}, &resp)

			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			if len(backend.viewSecretValue) != 1 || backend.viewSecretValue[0] != tc.wantView {
				t.Fatalf("expected one read with viewSecretValue=%s, got %v", tc.wantView, backend.viewSecretValue)
			}
			var value types.String
			var woVersion types.Int64
			resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("value"), &value)...)
			resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("value_wo_version"), &woVersion)...)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			if !value.Equal(tc.wantValue) || !woVersion.Equal(tc.wantWOVersion) {
				t.Errorf("expected value %v and value_wo_version %v, got %v and %v", tc.wantValue, tc.wantWOVersion, value, woVersion)
			}
		})
	}
}
