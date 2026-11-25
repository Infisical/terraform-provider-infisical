package resource

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const testADCSConnectionID = "3f091b42-1661-41a1-b647-98c6a45af047"

// Placeholder credentials for the fake backend.
const (
	testADCSFirstPassword  = "first-password"
	testADCSSecondPassword = "second-password"
)

// Fake of the routes that registerAppConnectionEndpoints in Infisical core registers for
// AppConnection.ADCS (backend/src/server/routes/v1/app-connection-routers/adcs-connection-router.ts).
// Like the real API, responses never contain credentials, only a hash that changes with them, and a
// missing connection is a 404.
type fakeADCSConnectionBackend struct {
	t           *testing.T
	app         string
	connections map[string]map[string]any
	credentials map[string]map[string]any
	// Connections with a credential rotation. Like the real GET by ID, responses don't include it.
	rotations map[string]bool
	requests  int
	lastBody  map[string]any
}

func newFakeADCSConnectionBackend(t *testing.T) *fakeADCSConnectionBackend {
	return &fakeADCSConnectionBackend{t: t, app: "adcs", connections: map[string]map[string]any{}, credentials: map[string]map[string]any{}, rotations: map[string]bool{}}
}

func (b *fakeADCSConnectionBackend) writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		b.t.Error(err)
	}
}

func credentialsHash(credentials map[string]any) string {
	raw, _ := json.Marshal(credentials)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (b *fakeADCSConnectionBackend) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b.requests++
		prefix := "/api/v1/app-connections/" + b.app
		if !strings.HasPrefix(r.URL.Path, prefix) {
			b.t.Errorf("unexpected path %s", r.URL.Path)
			b.writeJSON(w, http.StatusNotFound, map[string]any{"message": "not found"})
			return
		}
		id := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, prefix), "/")

		if r.Method == http.MethodPost || r.Method == http.MethodPatch {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				b.writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
				return
			}
			b.lastBody = body
			if r.Method == http.MethodPost {
				id = testADCSConnectionID
			} else if _, ok := b.connections[id]; !ok {
				b.writeJSON(w, http.StatusNotFound, map[string]any{"message": "not found"})
				return
			}
			existing := b.connections[id]
			credentials, credentialsSent := body["credentials"].(map[string]any)
			// updateAppConnection rejects credentials for a platform-managed connection, and
			// credentials sent to a connection with a rotation turn the rotation back on.
			if credentialsSent && existing["isPlatformManagedCredentials"] == true {
				b.writeJSON(w, http.StatusBadRequest, map[string]any{"message": "Cannot update credentials or management status for platform managed connections"})
				return
			}
			if credentialsSent {
				b.credentials[id] = credentials
			}
			gatewayId, gatewayPoolId := resolveGatewayFields(existing, body)
			b.connections[id] = map[string]any{
				"id":              id,
				"name":            body["name"],
				"description":     body["description"],
				"app":             b.app,
				"method":          body["method"],
				"projectId":       nil,
				"gatewayId":       gatewayId,
				"gatewayPoolId":   gatewayPoolId,
				"credentialsHash": credentialsHash(b.credentials[id]),
				"credentials":     sanitizedCredentials(b.credentials[id]),
			}
			for _, key := range []string{"isPlatformManagedCredentials", "isAutoRotationEnabled"} {
				if value, ok := existing[key]; ok {
					b.connections[id][key] = value
				}
			}
			if credentialsSent && b.rotations[id] {
				b.connections[id]["isAutoRotationEnabled"] = true
			}
			b.writeJSON(w, http.StatusOK, map[string]any{"appConnection": b.connections[id]})
			return
		}

		connection, ok := b.connections[id]
		if !ok {
			b.writeJSON(w, http.StatusNotFound, map[string]any{"message": "not found"})
			return
		}
		if r.Method == http.MethodDelete {
			delete(b.connections, id)
		}
		b.writeJSON(w, http.StatusOK, map[string]any{"appConnection": connection})
	})
}

// Like the real API, responses carry the non-secret credential fields and never the password.
func sanitizedCredentials(credentials map[string]any) map[string]any {
	sanitized := map[string]any{}
	for key, value := range credentials {
		if key != "password" {
			sanitized[key] = value
		}
	}
	return sanitized
}

// Mirrors the gateway mutual exclusion in updateAppConnection (Infisical core,
// backend/src/services/app-connection/app-connection-service.ts): an omitted field keeps its
// value, and setting one of gatewayId or gatewayPoolId clears the other only when the other was
// omitted.
func resolveGatewayFields(existing map[string]any, body map[string]any) (any, any) {
	var gatewayId, gatewayPoolId any
	if existing != nil {
		gatewayId, gatewayPoolId = existing["gatewayId"], existing["gatewayPoolId"]
	}
	sentGatewayId, gatewayIdSent := body["gatewayId"]
	sentPoolId, poolIdSent := body["gatewayPoolId"]
	switch {
	case gatewayIdSent && poolIdSent:
		gatewayId, gatewayPoolId = sentGatewayId, sentPoolId
	case gatewayIdSent:
		gatewayId = sentGatewayId
		if sentGatewayId != nil {
			gatewayPoolId = nil
		}
	case poolIdSent:
		gatewayPoolId = sentPoolId
		if sentPoolId != nil {
			gatewayId = nil
		}
	}
	return gatewayId, gatewayPoolId
}

func newADCSTestResource(t *testing.T, backend *fakeADCSConnectionBackend) *AppConnectionBaseResource {
	t.Helper()
	server := httptest.NewServer(backend.handler())
	t.Cleanup(server.Close)
	r, ok := NewAppConnectionADCSResource().(*AppConnectionBaseResource)
	if !ok {
		t.Fatal("NewAppConnectionADCSResource did not return the base resource")
	}
	r.client = &infisical.Client{Config: infisical.Config{
		HttpClient:            resty.New().SetBaseURL(server.URL),
		IsMachineIdentityAuth: true,
	}}
	return r
}

func adcsTestSchema(t *testing.T) schema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	NewAppConnectionADCSResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	return resp.Schema
}

var adcsCredentialsAttrTypes = map[string]attr.Type{
	"host":     types.StringType,
	"username": types.StringType,
	"password": types.StringType,
}

func adcsCredentials(t *testing.T, password string) types.Object {
	t.Helper()
	value, diags := types.ObjectValue(adcsCredentialsAttrTypes, map[string]attr.Value{
		"host":     types.StringValue("ca01.corp.example.com"),
		"username": types.StringValue(`CORP\svc-infisical`),
		"password": types.StringValue(password),
	})
	if diags.HasError() {
		t.Fatal(diags)
	}
	return value
}

func adcsPlan(t *testing.T, s schema.Schema, id types.String, password string) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: s}
	if diags := plan.Set(context.Background(), &appConnectionGatewayResourceModel{
		AppConnectionBaseResourceModel: AppConnectionBaseResourceModel{
			ID:              id,
			Name:            types.StringValue("adcs-connection"),
			Method:          types.StringValue(ADCSAppConnectionUsernamePasswordMethod),
			Description:     types.StringNull(),
			ProjectId:       types.StringNull(),
			Credentials:     adcsCredentials(t, password),
			CredentialsHash: types.StringUnknown(),
		},
		GatewayId: types.StringValue("gateway-1"),
	}); diags.HasError() {
		t.Fatal(diags)
	}
	return plan
}

func adcsModel(t *testing.T, state tfsdk.State) appConnectionGatewayResourceModel {
	t.Helper()
	var model appConnectionGatewayResourceModel
	if diags := state.Get(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return model
}

func TestADCSAppConnectionLifecycle(t *testing.T) {
	ctx := context.Background()
	s := adcsTestSchema(t)
	backend := newFakeADCSConnectionBackend(t)
	r := newADCSTestResource(t, backend)

	createResp := resource.CreateResponse{State: tfsdk.State{Schema: s}}
	r.Create(ctx, resource.CreateRequest{Plan: adcsPlan(t, s, types.StringUnknown(), testADCSFirstPassword)}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatal(createResp.Diagnostics)
	}

	wantCredentials := map[string]any{"host": "ca01.corp.example.com", "username": `CORP\svc-infisical`, "password": testADCSFirstPassword}
	if fmt.Sprint(backend.lastBody["credentials"]) != fmt.Sprint(wantCredentials) {
		t.Errorf("credentials sent = %v, want %v", backend.lastBody["credentials"], wantCredentials)
	}
	if backend.lastBody["gatewayId"] != "gateway-1" || backend.lastBody["method"] != "username-password" {
		t.Errorf("unexpected create body: %v", backend.lastBody)
	}

	created := adcsModel(t, createResp.State)
	if created.ID.ValueString() != testADCSConnectionID || created.CredentialsHash.ValueString() == "" {
		t.Fatalf("unexpected state after create: %+v", created)
	}

	readResp := resource.ReadResponse{State: createResp.State}
	r.Read(ctx, resource.ReadRequest{State: createResp.State}, &readResp)
	if readResp.Diagnostics.HasError() || readResp.Diagnostics.WarningsCount() != 0 {
		t.Fatalf("read after create should be clean, got %v", readResp.Diagnostics)
	}
	if !readResp.State.Raw.Equal(createResp.State.Raw) {
		t.Error("read after create changed state, which would show drift on the next plan")
	}

	updateResp := resource.UpdateResponse{State: createResp.State}
	r.Update(ctx, resource.UpdateRequest{Plan: adcsPlan(t, s, types.StringValue(testADCSConnectionID), testADCSSecondPassword), State: createResp.State}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatal(updateResp.Diagnostics)
	}
	if adcsModel(t, updateResp.State).CredentialsHash.Equal(created.CredentialsHash) {
		t.Error("expected the credentials hash to change after a password update")
	}

	deleteResp := resource.DeleteResponse{State: updateResp.State}
	r.Delete(ctx, resource.DeleteRequest{State: updateResp.State}, &deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatal(deleteResp.Diagnostics)
	}

	// The connection is gone now. A second delete gets a 404 and must still succeed.
	secondDeleteResp := resource.DeleteResponse{State: updateResp.State}
	r.Delete(ctx, resource.DeleteRequest{State: updateResp.State}, &secondDeleteResp)
	if secondDeleteResp.Diagnostics.HasError() {
		t.Fatalf("delete of a missing connection should succeed, got %v", secondDeleteResp.Diagnostics)
	}

	readResp = resource.ReadResponse{State: updateResp.State}
	r.Read(ctx, resource.ReadRequest{State: updateResp.State}, &readResp)
	if readResp.Diagnostics.HasError() || !readResp.State.Raw.IsNull() {
		t.Errorf("read of a missing connection should remove it from state, got %v", readResp.Diagnostics)
	}
}

func TestADCSAppConnectionReadDetectsOutOfBandCredentialChange(t *testing.T) {
	ctx := context.Background()
	s := adcsTestSchema(t)
	backend := newFakeADCSConnectionBackend(t)
	r := newADCSTestResource(t, backend)

	createResp := resource.CreateResponse{State: tfsdk.State{Schema: s}}
	r.Create(ctx, resource.CreateRequest{Plan: adcsPlan(t, s, types.StringUnknown(), testADCSFirstPassword)}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatal(createResp.Diagnostics)
	}

	backend.credentials[testADCSConnectionID] = map[string]any{"password": "changed-in-the-ui"}
	backend.connections[testADCSConnectionID]["credentialsHash"] = credentialsHash(backend.credentials[testADCSConnectionID])
	backend.connections[testADCSConnectionID]["name"] = "renamed-in-the-ui"

	readResp := resource.ReadResponse{State: createResp.State}
	r.Read(ctx, resource.ReadRequest{State: createResp.State}, &readResp)
	if readResp.Diagnostics.WarningsCount() != 1 {
		t.Errorf("expected one credentials conflict warning, got %v", readResp.Diagnostics)
	}
	model := adcsModel(t, readResp.State)
	var credentials AppConnectionADCSCredentialsModel
	if diags := model.Credentials.As(ctx, &credentials, basetypes.ObjectAsOptions{}); diags.HasError() {
		t.Fatal(diags)
	}
	if !credentials.Password.IsNull() {
		t.Error("expected credentials to be nulled so the next plan sends them again")
	}
	if model.Name.ValueString() != "renamed-in-the-ui" {
		t.Errorf("expected the other fields to still refresh, got %+v", model)
	}
}

func TestADCSAppConnectionImport(t *testing.T) {
	ctx := context.Background()
	s := adcsTestSchema(t)
	backend := newFakeADCSConnectionBackend(t)
	r := newADCSTestResource(t, backend)

	gatewayId := "gateway-1"
	backend.connections[testADCSConnectionID] = map[string]any{
		"id": testADCSConnectionID, "name": "adcs-connection", "description": "", "app": "adcs",
		"method": "username-password", "projectId": nil, "gatewayId": gatewayId, "credentialsHash": "hash-1",
	}

	emptyState := tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}

	badResp := resource.ImportStateResponse{State: emptyState}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "adcs-connection"}, &badResp)
	if !badResp.Diagnostics.HasError() {
		t.Error("expected an error for an import ID that is not a UUID")
	}

	importResp := resource.ImportStateResponse{State: emptyState}
	r.ImportState(ctx, resource.ImportStateRequest{ID: testADCSConnectionID}, &importResp)
	if importResp.Diagnostics.HasError() {
		t.Fatal(importResp.Diagnostics)
	}

	readResp := resource.ReadResponse{State: importResp.State}
	r.Read(ctx, resource.ReadRequest{State: importResp.State}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatal(readResp.Diagnostics)
	}
	if readResp.Diagnostics.WarningsCount() != 0 {
		t.Errorf("an import has no hash to conflict with, so it must not warn, got %v", readResp.Diagnostics)
	}

	model := adcsModel(t, readResp.State)
	if model.Name.ValueString() != "adcs-connection" || model.Method.ValueString() != "username-password" ||
		model.GatewayId.ValueString() != gatewayId || model.CredentialsHash.ValueString() != "hash-1" ||
		!model.Description.IsNull() || !model.ProjectId.IsNull() {
		t.Errorf("import did not fill state from the API: %+v", model)
	}
}

func TestADCSAppConnectionReadWithEmptyIDRemovesResource(t *testing.T) {
	ctx := context.Background()
	s := adcsTestSchema(t)
	backend := newFakeADCSConnectionBackend(t)
	r := newADCSTestResource(t, backend)

	plan := adcsPlan(t, s, types.StringValue(""), "password")
	state := tfsdk.State{Schema: s, Raw: plan.Raw}
	readResp := resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, &readResp)
	if readResp.Diagnostics.HasError() || !readResp.State.Raw.IsNull() {
		t.Errorf("expected the resource to be removed, got %v", readResp.Diagnostics)
	}
	if backend.requests != 0 {
		t.Errorf("an empty ID must not reach the API, got %d requests", backend.requests)
	}
}

func TestADCSAppConnectionSchema(t *testing.T) {
	s := adcsTestSchema(t)

	for _, name := range []string{"gateway_id", "gateway_pool_id"} {
		if attribute, ok := s.Attributes[name].(schema.StringAttribute); !ok || !attribute.Optional {
			t.Errorf("%s must be optional, the config validator requires exactly one of them", name)
		}
	}

	credentials, ok := s.Attributes["credentials"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatal("credentials must be a single nested attribute")
	}
	password, ok := credentials.Attributes["password"].(schema.StringAttribute)
	if !ok || !password.Sensitive {
		t.Error("password must be sensitive")
	}
}

// Import lives in the shared base, so it must also work for a variant without gateway support,
// whose state has no gateway_id.
func TestAppConnectionImportOnVariantWithoutGateway(t *testing.T) {
	ctx := context.Background()
	backend := newFakeADCSConnectionBackend(t)
	backend.app = "render"
	server := httptest.NewServer(backend.handler())
	t.Cleanup(server.Close)

	r, ok := NewAppConnectionRenderResource().(*AppConnectionBaseResource)
	if !ok {
		t.Fatal("NewAppConnectionRenderResource did not return the base resource")
	}
	r.client = &infisical.Client{Config: infisical.Config{
		HttpClient:            resty.New().SetBaseURL(server.URL),
		IsMachineIdentityAuth: true,
	}}

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema

	backend.connections[testADCSConnectionID] = map[string]any{
		"id": testADCSConnectionID, "name": "render-connection", "description": "Deploys", "app": "render",
		"method": "api-key", "projectId": "project-1", "credentialsHash": "hash-1",
	}

	importResp := resource.ImportStateResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}}
	r.ImportState(ctx, resource.ImportStateRequest{ID: testADCSConnectionID}, &importResp)
	if importResp.Diagnostics.HasError() {
		t.Fatal(importResp.Diagnostics)
	}

	readResp := resource.ReadResponse{State: importResp.State}
	r.Read(ctx, resource.ReadRequest{State: importResp.State}, &readResp)
	if readResp.Diagnostics.HasError() || readResp.Diagnostics.WarningsCount() != 0 {
		t.Fatalf("expected a clean read after import, got %v", readResp.Diagnostics)
	}

	var model AppConnectionBaseResourceModel
	if diags := readResp.State.Get(ctx, &model); diags.HasError() {
		t.Fatal(diags)
	}
	if model.Name.ValueString() != "render-connection" || model.Method.ValueString() != "api-key" ||
		model.Description.ValueString() != "Deploys" || model.ProjectId.ValueString() != "project-1" {
		t.Errorf("import did not fill state from the API: %+v", model)
	}
}
