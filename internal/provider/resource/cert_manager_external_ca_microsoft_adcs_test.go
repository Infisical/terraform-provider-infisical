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
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const (
	testMicrosoftADCSCAID         = "6d1e5a2b-8c3f-4e7a-9b10-2c3d4e5f6a7b"
	testMicrosoftADCSConnectionID = "3f091b42-1661-41a1-b647-98c6a45af047"
	testOtherADCSConnectionID     = "9a8b7c6d-5e4f-4a3b-8c2d-1e0f9a8b7c6d"
	testDiscoveredCaName          = "corp-ca01-CA"
)

// Fake of the "adcs" CA routes in Infisical core
// (backend/src/server/routes/v1/certificate-authority-routers/adcs-certificate-authority-router.ts and
// backend/src/services/certificate-authority/adcs/adcs-certificate-authority-fns.ts). Like the real
// API it trims caName, discovers the name from the CA host when caName is omitted (on create, and on
// an update that sends configuration without it), and answers 404 for a missing CA.
type fakeMicrosoftADCSCABackend struct {
	t           *testing.T
	cas         map[string]infisical.MicrosoftADCSCA
	discoveries int
	requests    int
	lastBody    map[string]any
}

func newFakeMicrosoftADCSCABackend(t *testing.T) *fakeMicrosoftADCSCABackend {
	return &fakeMicrosoftADCSCABackend{t: t, cas: map[string]infisical.MicrosoftADCSCA{}}
}

func (b *fakeMicrosoftADCSCABackend) writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		b.t.Error(err)
	}
}

func (b *fakeMicrosoftADCSCABackend) resolveCaName(raw any) *string {
	if name, ok := raw.(string); ok && strings.TrimSpace(name) != "" {
		trimmed := strings.TrimSpace(name)
		return &trimmed
	}
	b.discoveries++
	discovered := testDiscoveredCaName
	return &discovered
}

func (b *fakeMicrosoftADCSCABackend) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b.requests++
		const prefix = "/api/v1/cert-manager/ca/adcs"
		if !strings.HasPrefix(r.URL.Path, prefix) {
			b.t.Errorf("unexpected path %s", r.URL.Path)
			b.writeJSON(w, http.StatusNotFound, map[string]any{"message": "not found"})
			return
		}
		id := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, prefix), "/")

		var body map[string]any
		if r.Method == http.MethodPost || r.Method == http.MethodPatch {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				b.writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
				return
			}
			b.lastBody = body
		}

		if r.Method == http.MethodPost {
			configuration, _ := body["configuration"].(map[string]any)
			connectionId, _ := configuration["appConnectionId"].(string)
			name, _ := body["name"].(string)
			status, _ := body["status"].(string)
			ca := infisical.MicrosoftADCSCA{
				Id: testMicrosoftADCSCAID, ProjectId: "project-1", Name: name, Status: status,
				Configuration: infisical.MicrosoftADCSCAConfiguration{AppConnectionId: connectionId, CaName: b.resolveCaName(configuration["caName"])},
			}
			b.cas[ca.Id] = ca
			b.writeJSON(w, http.StatusOK, ca)
			return
		}

		ca, ok := b.cas[id]
		if !ok {
			b.writeJSON(w, http.StatusNotFound, map[string]any{"message": "Certificate Authority not found"})
			return
		}

		switch r.Method {
		case http.MethodPatch:
			if name, ok := body["name"].(string); ok && name != "" {
				ca.Name = name
			}
			if status, ok := body["status"].(string); ok && status != "" {
				ca.Status = status
			}
			if configuration, ok := body["configuration"].(map[string]any); ok {
				connectionId, _ := configuration["appConnectionId"].(string)
				ca.Configuration = infisical.MicrosoftADCSCAConfiguration{AppConnectionId: connectionId, CaName: b.resolveCaName(configuration["caName"])}
			}
			b.cas[id] = ca
		case http.MethodDelete:
			delete(b.cas, id)
		}
		b.writeJSON(w, http.StatusOK, ca)
	})
}

func newMicrosoftADCSCATestResource(t *testing.T, backend *fakeMicrosoftADCSCABackend) *certManagerExternalCAMicrosoftADCSResource {
	t.Helper()
	server := httptest.NewServer(backend.handler())
	t.Cleanup(server.Close)
	return &certManagerExternalCAMicrosoftADCSResource{client: &infisical.Client{Config: infisical.Config{
		HttpClient:            resty.New().SetBaseURL(server.URL),
		IsMachineIdentityAuth: true,
	}}}
}

func microsoftADCSCATestSchema(t *testing.T) schema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	NewCertManagerExternalCAMicrosoftADCSResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	return resp.Schema
}

func microsoftADCSCAPlan(t *testing.T, s schema.Schema, model certManagerExternalCAMicrosoftADCSResourceModel) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: s}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return plan
}

func microsoftADCSCAModel(t *testing.T, state tfsdk.State) certManagerExternalCAMicrosoftADCSResourceModel {
	t.Helper()
	var model certManagerExternalCAMicrosoftADCSResourceModel
	if diags := state.Get(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return model
}

func TestMicrosoftADCSCALifecycleWithDiscoveredCaName(t *testing.T) {
	ctx := context.Background()
	s := microsoftADCSCATestSchema(t)
	backend := newFakeMicrosoftADCSCABackend(t)
	r := newMicrosoftADCSCATestResource(t, backend)

	createResp := resource.CreateResponse{State: tfsdk.State{Schema: s}}
	r.Create(ctx, resource.CreateRequest{Plan: microsoftADCSCAPlan(t, s, certManagerExternalCAMicrosoftADCSResourceModel{
		Id:              types.StringUnknown(),
		Name:            types.StringValue("corporate-adcs"),
		Status:          types.StringValue("active"),
		AppConnectionId: types.StringValue(testMicrosoftADCSConnectionID),
		CaName:          types.StringUnknown(),
	})}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatal(createResp.Diagnostics)
	}

	configuration, _ := backend.lastBody["configuration"].(map[string]any)
	if _, sent := configuration["caName"]; sent {
		t.Errorf("an unset ca_name must be left out so the API discovers it, got body %v", backend.lastBody)
	}
	if configuration["appConnectionId"] != testMicrosoftADCSConnectionID || backend.lastBody["status"] != "active" || backend.lastBody["name"] != "corporate-adcs" {
		t.Errorf("unexpected create body: %v", backend.lastBody)
	}

	created := microsoftADCSCAModel(t, createResp.State)
	if created.Id.ValueString() != testMicrosoftADCSCAID || created.CaName.ValueString() != testDiscoveredCaName {
		t.Fatalf("unexpected state after create: %+v", created)
	}

	readResp := resource.ReadResponse{State: createResp.State}
	r.Read(ctx, resource.ReadRequest{State: createResp.State}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatal(readResp.Diagnostics)
	}
	if !readResp.State.Raw.Equal(createResp.State.Raw) {
		t.Error("read after create changed state, which would show drift on the next plan")
	}

	// A status change keeps ca_name from state (see caNameUseStateUnlessConnectionChanges). The
	// update must leave configuration out, because sending it needs permission to use the connection.
	updatePlan := created
	updatePlan.Status = types.StringValue("disabled")
	updateResp := resource.UpdateResponse{State: createResp.State}
	r.Update(ctx, resource.UpdateRequest{Plan: microsoftADCSCAPlan(t, s, updatePlan), State: createResp.State}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatal(updateResp.Diagnostics)
	}
	if _, sent := backend.lastBody["configuration"]; sent {
		t.Errorf("a status change must not send configuration, got body %v", backend.lastBody)
	}
	if backend.discoveries != 1 {
		t.Errorf("expected discovery only on create, got %d discoveries", backend.discoveries)
	}
	if got := microsoftADCSCAModel(t, updateResp.State); got.Status.ValueString() != "disabled" || got.CaName.ValueString() != testDiscoveredCaName {
		t.Errorf("unexpected state after update: %+v", got)
	}

	// A new connection with ca_name unset plans ca_name as unknown, so the API discovers it again.
	connectionPlan := microsoftADCSCAModel(t, updateResp.State)
	connectionPlan.AppConnectionId = types.StringValue(testOtherADCSConnectionID)
	connectionPlan.CaName = types.StringUnknown()
	connectionResp := resource.UpdateResponse{State: updateResp.State}
	r.Update(ctx, resource.UpdateRequest{Plan: microsoftADCSCAPlan(t, s, connectionPlan), State: updateResp.State}, &connectionResp)
	if connectionResp.Diagnostics.HasError() {
		t.Fatal(connectionResp.Diagnostics)
	}
	configuration, _ = backend.lastBody["configuration"].(map[string]any)
	if configuration["appConnectionId"] != testOtherADCSConnectionID {
		t.Errorf("a connection change must send configuration, got body %v", backend.lastBody)
	}
	if _, sent := configuration["caName"]; sent || backend.discoveries != 2 {
		t.Errorf("a connection change with ca_name unset must leave caName out so the API discovers it, got body %v and %d discoveries", backend.lastBody, backend.discoveries)
	}
	updateResp = connectionResp

	deleteResp := resource.DeleteResponse{State: updateResp.State}
	r.Delete(ctx, resource.DeleteRequest{State: updateResp.State}, &deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatal(deleteResp.Diagnostics)
	}
	if _, exists := backend.cas[testMicrosoftADCSCAID]; exists {
		t.Fatal("expected the CA to be deleted on the backend")
	}

	secondDeleteResp := resource.DeleteResponse{State: updateResp.State}
	r.Delete(ctx, resource.DeleteRequest{State: updateResp.State}, &secondDeleteResp)
	if secondDeleteResp.Diagnostics.HasError() {
		t.Errorf("delete of a missing CA should succeed, got %v", secondDeleteResp.Diagnostics)
	}

	goneResp := resource.ReadResponse{State: updateResp.State}
	r.Read(ctx, resource.ReadRequest{State: updateResp.State}, &goneResp)
	if goneResp.Diagnostics.HasError() || !goneResp.State.Raw.IsNull() {
		t.Errorf("read of a missing CA should remove it from state, got %v", goneResp.Diagnostics)
	}
}

func TestMicrosoftADCSCAKeepsConfiguredCaNameWhenOnlyWhitespaceDiffers(t *testing.T) {
	ctx := context.Background()
	s := microsoftADCSCATestSchema(t)
	backend := newFakeMicrosoftADCSCABackend(t)
	r := newMicrosoftADCSCATestResource(t, backend)

	createResp := resource.CreateResponse{State: tfsdk.State{Schema: s}}
	r.Create(ctx, resource.CreateRequest{Plan: microsoftADCSCAPlan(t, s, certManagerExternalCAMicrosoftADCSResourceModel{
		Id:              types.StringUnknown(),
		Name:            types.StringValue("corporate-adcs"),
		Status:          types.StringValue("active"),
		AppConnectionId: types.StringValue(testMicrosoftADCSConnectionID),
		CaName:          types.StringValue(" corp-ca01-CA "),
	})}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatal(createResp.Diagnostics)
	}
	if backend.discoveries != 0 {
		t.Error("a configured ca_name must not trigger discovery")
	}

	readResp := resource.ReadResponse{State: createResp.State}
	r.Read(ctx, resource.ReadRequest{State: createResp.State}, &readResp)
	if got := microsoftADCSCAModel(t, readResp.State).CaName.ValueString(); got != " corp-ca01-CA " {
		t.Errorf("expected the configured spelling to survive the API's trim, got %q", got)
	}
}

func TestMicrosoftADCSCAReadWithEmptyIDRemovesResource(t *testing.T) {
	ctx := context.Background()
	s := microsoftADCSCATestSchema(t)
	backend := newFakeMicrosoftADCSCABackend(t)
	r := newMicrosoftADCSCATestResource(t, backend)

	plan := microsoftADCSCAPlan(t, s, certManagerExternalCAMicrosoftADCSResourceModel{
		Id:              types.StringValue(""),
		Name:            types.StringValue("corporate-adcs"),
		Status:          types.StringValue("active"),
		AppConnectionId: types.StringValue(testMicrosoftADCSConnectionID),
		CaName:          types.StringNull(),
	})
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

func TestMicrosoftADCSCAImport(t *testing.T) {
	ctx := context.Background()
	s := microsoftADCSCATestSchema(t)
	backend := newFakeMicrosoftADCSCABackend(t)
	r := newMicrosoftADCSCATestResource(t, backend)

	caName := testDiscoveredCaName
	backend.cas[testMicrosoftADCSCAID] = infisical.MicrosoftADCSCA{
		Id: testMicrosoftADCSCAID, ProjectId: "project-1", Name: "corporate-adcs", Status: "active",
		Configuration: infisical.MicrosoftADCSCAConfiguration{AppConnectionId: testMicrosoftADCSConnectionID, CaName: &caName},
	}

	emptyState := tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}

	badResp := resource.ImportStateResponse{State: emptyState}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "corporate-adcs"}, &badResp)
	if !badResp.Diagnostics.HasError() {
		t.Error("expected an error for an import ID that is not a UUID")
	}

	importResp := resource.ImportStateResponse{State: emptyState}
	r.ImportState(ctx, resource.ImportStateRequest{ID: testMicrosoftADCSCAID}, &importResp)
	if importResp.Diagnostics.HasError() {
		t.Fatal(importResp.Diagnostics)
	}

	readResp := resource.ReadResponse{State: importResp.State}
	r.Read(ctx, resource.ReadRequest{State: importResp.State}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatal(readResp.Diagnostics)
	}

	got := microsoftADCSCAModel(t, readResp.State)
	want := certManagerExternalCAMicrosoftADCSResourceModel{
		Id:              types.StringValue(testMicrosoftADCSCAID),
		Name:            types.StringValue("corporate-adcs"),
		Status:          types.StringValue("active"),
		AppConnectionId: types.StringValue(testMicrosoftADCSConnectionID),
		CaName:          types.StringValue(testDiscoveredCaName),
	}
	if got != want {
		t.Errorf("import state = %+v, want %+v", got, want)
	}
}

func TestMicrosoftADCSCANamePlanModifier(t *testing.T) {
	ctx := context.Background()
	s := microsoftADCSCATestSchema(t)

	stateModel := certManagerExternalCAMicrosoftADCSResourceModel{
		Id:              types.StringValue(testMicrosoftADCSCAID),
		Name:            types.StringValue("corporate-adcs"),
		Status:          types.StringValue("active"),
		AppConnectionId: types.StringValue(testMicrosoftADCSConnectionID),
		CaName:          types.StringValue(testDiscoveredCaName),
	}
	state := tfsdk.State{Schema: s, Raw: microsoftADCSCAPlan(t, s, stateModel).Raw}

	cases := []struct {
		name         string
		connectionId string
		configValue  types.String
		planValue    types.String
		want         types.String
	}{
		{"unset, same connection keeps the prior name", testMicrosoftADCSConnectionID, types.StringNull(), types.StringUnknown(), types.StringValue(testDiscoveredCaName)},
		{"unset, new connection is discovered again", testOtherADCSConnectionID, types.StringNull(), types.StringUnknown(), types.StringUnknown()},
		{"configured value wins", testMicrosoftADCSConnectionID, types.StringValue("other-CA"), types.StringValue("other-CA"), types.StringValue("other-CA")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			planModel := stateModel
			planModel.AppConnectionId = types.StringValue(tc.connectionId)
			planModel.CaName = tc.planValue
			plan := microsoftADCSCAPlan(t, s, planModel)

			req := planmodifier.StringRequest{
				Path:        path.Root("ca_name"),
				ConfigValue: tc.configValue,
				PlanValue:   tc.planValue,
				StateValue:  stateModel.CaName,
				Plan:        plan,
				State:       state,
			}
			resp := planmodifier.StringResponse{PlanValue: tc.planValue}
			caNameUseStateUnlessConnectionChanges{}.PlanModifyString(ctx, req, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			if !resp.PlanValue.Equal(tc.want) {
				t.Errorf("plan value = %s, want %s", resp.PlanValue, tc.want)
			}
		})
	}
}

func TestMicrosoftADCSCAReadSurfacesServerErrors(t *testing.T) {
	ctx := context.Background()
	s := microsoftADCSCATestSchema(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"You are not allowed to read certificate authorities"}`))
	}))
	t.Cleanup(server.Close)
	r := &certManagerExternalCAMicrosoftADCSResource{client: &infisical.Client{Config: infisical.Config{
		HttpClient:            resty.New().SetBaseURL(server.URL),
		IsMachineIdentityAuth: true,
	}}}

	plan := microsoftADCSCAPlan(t, s, certManagerExternalCAMicrosoftADCSResourceModel{
		Id:              types.StringValue(testMicrosoftADCSCAID),
		Name:            types.StringValue("corporate-adcs"),
		Status:          types.StringValue("active"),
		AppConnectionId: types.StringValue(testMicrosoftADCSConnectionID),
		CaName:          types.StringValue(testDiscoveredCaName),
	})
	state := tfsdk.State{Schema: s, Raw: plan.Raw}
	readResp := resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, &readResp)
	if !readResp.Diagnostics.HasError() {
		t.Error("a 403 must be an error, not a silent removal from state")
	}
}

// The API trims ca_name before its minimum length check, so whitespace alone must fail at plan.
func TestMicrosoftADCSCARejectsWhitespaceOnlyCaName(t *testing.T) {
	ctx := context.Background()
	caName, ok := microsoftADCSCATestSchema(t).Attributes["ca_name"].(schema.StringAttribute)
	if !ok {
		t.Fatal("ca_name must be a string attribute")
	}

	for value, wantError := range map[string]bool{"": true, "   ": true, " corp-ca01-CA ": false} {
		resp := validator.StringResponse{}
		for _, v := range caName.Validators {
			v.ValidateString(ctx, validator.StringRequest{Path: path.Root("ca_name"), ConfigValue: types.StringValue(value)}, &resp)
		}
		if resp.Diagnostics.HasError() != wantError {
			t.Errorf("ca_name %q: error = %v, want %v", value, resp.Diagnostics.HasError(), wantError)
		}
	}
}
