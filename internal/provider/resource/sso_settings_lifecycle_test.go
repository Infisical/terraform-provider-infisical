package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const ssoSettingsTestOrgID = "8a1f2b3c-4d5e-4f60-9a7b-8c9d0e1f2a3b"

type fakeSsoSettings struct {
	OrgID                 string `json:"orgId"`
	AuthEnforced          bool   `json:"authEnforced"`
	GoogleSsoAuthEnforced bool   `json:"googleSsoAuthEnforced"`
	BypassOrgAuthEnabled  bool   `json:"bypassOrgAuthEnabled"`
	SsoMfaTrust           string `json:"ssoMfaTrust"`
}

type fakeSsoSettingsBackend struct {
	t           *testing.T
	settings    *fakeSsoSettings
	oidcActive  bool
	oidcUsed    bool
	googleUsed  bool
	patchBodies []map[string]any
}

func newFakeSsoSettingsBackend(t *testing.T) *fakeSsoSettingsBackend {
	return &fakeSsoSettingsBackend{
		t:        t,
		settings: &fakeSsoSettings{OrgID: ssoSettingsTestOrgID, SsoMfaTrust: "none"},
	}
}

func (b *fakeSsoSettingsBackend) writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		b.t.Error(err)
	}
}

func (b *fakeSsoSettingsBackend) apiError(w http.ResponseWriter, status int, message string) {
	b.writeJSON(w, status, map[string]any{"statusCode": status, "message": message})
}

// a change to a connection field of an active OIDC configuration turns enforcement off and clears lastUsed.
func (b *fakeSsoSettingsBackend) changeOidcConnection() {
	b.settings.AuthEnforced = false
	b.oidcUsed = false
}

func (b *fakeSsoSettingsBackend) handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/organization/sso-settings", func(w http.ResponseWriter, _ *http.Request) {
		if b.settings == nil {
			b.apiError(w, http.StatusNotFound, fmt.Sprintf("Organization with ID '%s' not found", ssoSettingsTestOrgID))
			return
		}
		b.writeJSON(w, http.StatusOK, map[string]any{"ssoSettings": b.settings})
	})

	mux.HandleFunc("PATCH /api/v1/organization/sso-settings", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			b.apiError(w, http.StatusUnprocessableEntity, "Invalid body")
			return
		}
		b.patchBodies = append(b.patchBodies, body)
		if len(body) == 0 {
			b.apiError(w, http.StatusUnprocessableEntity, "Provide at least one SSO setting to update")
			return
		}
		if b.settings == nil {
			b.apiError(w, http.StatusNotFound, fmt.Sprintf("Organization with ID '%s' not found", ssoSettingsTestOrgID))
			return
		}

		authEnforced, hasAuth := body["authEnforced"].(bool)
		googleEnforced, hasGoogle := body["googleSsoAuthEnforced"].(bool)

		willEnforceSso := b.settings.AuthEnforced
		if hasAuth {
			willEnforceSso = authEnforced
		}
		willEnforceGoogle := b.settings.GoogleSsoAuthEnforced
		if hasGoogle {
			willEnforceGoogle = googleEnforced
		}
		if willEnforceSso && willEnforceGoogle && (authEnforced || googleEnforced) {
			b.apiError(w, http.StatusBadRequest, "SAML/OIDC auth enforcement and Google SSO auth enforcement cannot be enabled at the same time.")
			return
		}
		if authEnforced {
			if !b.oidcActive {
				b.apiError(w, http.StatusBadRequest, "Cannot enforce SSO because the organization has no active SAML or OIDC configuration.")
				return
			}
			if !b.oidcUsed {
				b.apiError(w, http.StatusBadRequest, "To apply the new OIDC auth enforcement, please log in via OIDC at least once. This step is required to enforce OIDC-based authentication.")
				return
			}
		}
		if googleEnforced {
			if b.oidcActive {
				b.apiError(w, http.StatusBadRequest, "Cannot enable Google OAuth enforcement while OIDC SSO is configured. Disable OIDC SSO to enforce Google OAuth.")
				return
			}
			if !b.googleUsed {
				b.apiError(w, http.StatusBadRequest, "Google SSO auth enforcement cannot be enabled because Google SSO has not been used yet. Please log in via Google SSO at least once before enforcing it for your organization.")
				return
			}
		}

		if hasAuth {
			b.settings.AuthEnforced = authEnforced
		}
		if hasGoogle {
			b.settings.GoogleSsoAuthEnforced = googleEnforced
		}
		if bypass, ok := body["bypassOrgAuthEnabled"].(bool); ok {
			b.settings.BypassOrgAuthEnabled = bypass
		}
		if trust, ok := body["ssoMfaTrust"].(string); ok {
			b.settings.SsoMfaTrust = trust
		}
		b.writeJSON(w, http.StatusOK, map[string]any{"ssoSettings": b.settings})
	})

	return mux
}

func newSsoSettingsTestResource(t *testing.T, backend *fakeSsoSettingsBackend) *SsoSettingsResource {
	t.Helper()
	server := httptest.NewServer(backend.handler())
	t.Cleanup(server.Close)
	return &SsoSettingsResource{client: &infisical.Client{Config: infisical.Config{
		HttpClient:            resty.New().SetBaseURL(server.URL),
		IsMachineIdentityAuth: true,
	}}}
}

func ssoSettingsTestSchema(t *testing.T) schema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	NewSsoSettingsResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func ssoSettingsDefaultModel() SsoSettingsResourceModel {
	return SsoSettingsResourceModel{
		ID:                    types.StringUnknown(),
		EnforceSso:            types.BoolValue(false),
		EnforceGoogleSso:      types.BoolValue(false),
		AdminSsoBypassEnabled: types.BoolValue(false),
		SsoMfaTrust:           types.StringValue("none"),
	}
}

func ssoSettingsState(t *testing.T, s schema.Schema, model SsoSettingsResourceModel) tfsdk.State {
	t.Helper()
	state := tfsdk.State{Schema: s}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return state
}

func ssoSettingsPlan(t *testing.T, s schema.Schema, model SsoSettingsResourceModel) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: s}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return plan
}

func ssoSettingsModel(t *testing.T, state tfsdk.State) SsoSettingsResourceModel {
	t.Helper()
	var model SsoSettingsResourceModel
	if diags := state.Get(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return model
}

func createSsoSettings(r *SsoSettingsResource, plan tfsdk.Plan) resource.CreateResponse {
	resp := resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema}}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, &resp)
	return resp
}

func readSsoSettings(r *SsoSettingsResource, state tfsdk.State) resource.ReadResponse {
	resp := resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
	return resp
}

func TestSsoSettingsLifecycle(t *testing.T) {
	s := ssoSettingsTestSchema(t)
	backend := newFakeSsoSettingsBackend(t)
	backend.oidcActive, backend.oidcUsed = true, true
	r := newSsoSettingsTestResource(t, backend)

	model := ssoSettingsDefaultModel()
	model.EnforceSso = types.BoolValue(true)
	model.SsoMfaTrust = types.StringValue("signal")
	createResp := createSsoSettings(r, ssoSettingsPlan(t, s, model))
	if createResp.Diagnostics.HasError() {
		t.Fatal(createResp.Diagnostics)
	}

	want := map[string]any{
		"authEnforced":          true,
		"googleSsoAuthEnforced": false,
		"bypassOrgAuthEnabled":  false,
		"ssoMfaTrust":           "signal",
	}
	if !reflect.DeepEqual(backend.patchBodies[0], want) {
		t.Errorf("unexpected create body:\n got: %v\nwant: %v", backend.patchBodies[0], want)
	}

	created := ssoSettingsModel(t, createResp.State)
	wantModel := SsoSettingsResourceModel{
		ID:                    types.StringValue(ssoSettingsTestOrgID),
		EnforceSso:            types.BoolValue(true),
		EnforceGoogleSso:      types.BoolValue(false),
		AdminSsoBypassEnabled: types.BoolValue(false),
		SsoMfaTrust:           types.StringValue("signal"),
	}
	if !reflect.DeepEqual(created, wantModel) {
		t.Errorf("unexpected state after create: %+v", created)
	}

	readResp := readSsoSettings(r, createResp.State)
	if readResp.Diagnostics.HasError() {
		t.Fatal(readResp.Diagnostics)
	}
	if refreshed := ssoSettingsModel(t, readResp.State); !reflect.DeepEqual(refreshed, created) {
		t.Errorf("expected no drift after a refresh, got %+v", refreshed)
	}

	// Turning enforcement off must send false, not leave the field out.
	model = created
	model.EnforceSso = types.BoolValue(false)
	model.AdminSsoBypassEnabled = types.BoolValue(true)
	updateResp := resource.UpdateResponse{State: createResp.State}
	r.Update(context.Background(), resource.UpdateRequest{State: createResp.State, Plan: ssoSettingsPlan(t, s, model)}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatal(updateResp.Diagnostics)
	}
	if got := backend.patchBodies[1]; got["authEnforced"] != false || got["bypassOrgAuthEnabled"] != true || got["ssoMfaTrust"] != "signal" {
		t.Errorf("unexpected update body: %v", got)
	}
	if updated := ssoSettingsModel(t, updateResp.State); !reflect.DeepEqual(updated, model) {
		t.Errorf("unexpected state after update: %+v", updated)
	}

	var deleteResp resource.DeleteResponse
	r.Delete(context.Background(), resource.DeleteRequest{State: updateResp.State}, &deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatal(deleteResp.Diagnostics)
	}
	wantDefaults := fakeSsoSettings{OrgID: ssoSettingsTestOrgID, SsoMfaTrust: "none"}
	if *backend.settings != wantDefaults {
		t.Errorf("expected delete to restore the defaults, got %+v", *backend.settings)
	}
}

func TestSsoSettingsReadRejectsOtherOrganization(t *testing.T) {
	s := ssoSettingsTestSchema(t)
	backend := newFakeSsoSettingsBackend(t)
	r := newSsoSettingsTestResource(t, backend)

	model := ssoSettingsDefaultModel()
	model.ID = types.StringValue(ssoSettingsTestOrgID)
	backend.settings.OrgID = "00000000-0000-4000-8000-000000000002"
	readResp := readSsoSettings(r, ssoSettingsState(t, s, model))
	if !readResp.Diagnostics.HasError() || !strings.Contains(fmt.Sprint(readResp.Diagnostics), "belong to a different organization") {
		t.Errorf("expected a mismatch error, got: %v", readResp.Diagnostics)
	}
}

// Infisical rejects enforcement until someone has logged in through the active configuration.
func TestSsoSettingsCreateEnforceBeforeLogin(t *testing.T) {
	s := ssoSettingsTestSchema(t)
	backend := newFakeSsoSettingsBackend(t)
	backend.oidcActive = true
	r := newSsoSettingsTestResource(t, backend)

	model := ssoSettingsDefaultModel()
	model.EnforceSso = types.BoolValue(true)
	resp := createSsoSettings(r, ssoSettingsPlan(t, s, model))
	if diags := fmt.Sprint(resp.Diagnostics); !strings.Contains(diags, "log in via OIDC at least once") {
		t.Errorf("expected the API's login error, got: %v", diags)
	}
	if backend.settings.AuthEnforced {
		t.Error("expected enforcement to stay off")
	}
}

// A change to the OIDC connection turns enforcement off in Infisical. Read must show it as drift.
func TestSsoSettingsReadShowsEnforcementResetByOidcChange(t *testing.T) {
	s := ssoSettingsTestSchema(t)
	backend := newFakeSsoSettingsBackend(t)
	backend.oidcActive, backend.oidcUsed = true, true
	r := newSsoSettingsTestResource(t, backend)

	model := ssoSettingsDefaultModel()
	model.EnforceSso = types.BoolValue(true)
	createResp := createSsoSettings(r, ssoSettingsPlan(t, s, model))
	if createResp.Diagnostics.HasError() {
		t.Fatal(createResp.Diagnostics)
	}

	backend.changeOidcConnection()

	readResp := readSsoSettings(r, createResp.State)
	if readResp.Diagnostics.HasError() {
		t.Fatal(readResp.Diagnostics)
	}
	if ssoSettingsModel(t, readResp.State).EnforceSso.ValueBool() {
		t.Error("expected Read to show that enforcement is off")
	}
}

func TestSsoSettingsReadNotFound(t *testing.T) {
	s := ssoSettingsTestSchema(t)
	backend := newFakeSsoSettingsBackend(t)
	backend.settings = nil
	r := newSsoSettingsTestResource(t, backend)

	model := ssoSettingsDefaultModel()
	model.ID = types.StringValue(ssoSettingsTestOrgID)
	resp := readSsoSettings(r, ssoSettingsState(t, s, model))
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Error("expected a 404 to remove the resource from state")
	}
}

func TestSsoSettingsReadEmptyID(t *testing.T) {
	s := ssoSettingsTestSchema(t)
	backend := newFakeSsoSettingsBackend(t)
	r := newSsoSettingsTestResource(t, backend)

	model := ssoSettingsDefaultModel()
	model.ID = types.StringValue("")
	resp := readSsoSettings(r, ssoSettingsState(t, s, model))
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Error("expected an empty ID to remove the resource from state")
	}
}

func TestSsoSettingsImport(t *testing.T) {
	ctx := context.Background()
	s := ssoSettingsTestSchema(t)
	backend := newFakeSsoSettingsBackend(t)
	backend.settings.BypassOrgAuthEnabled = true
	backend.settings.SsoMfaTrust = "signal"
	r := newSsoSettingsTestResource(t, backend)

	importResp := func(id string) resource.ImportStateResponse {
		resp := resource.ImportStateResponse{State: ssoSettingsState(t, s, ssoSettingsDefaultModel())}
		r.ImportState(ctx, resource.ImportStateRequest{ID: id}, &resp)
		return resp
	}

	if diags := fmt.Sprint(importResp("not-a-uuid").Diagnostics); !strings.Contains(diags, "Invalid import ID") {
		t.Errorf("expected an Invalid import ID error, got: %v", diags)
	}
	if diags := fmt.Sprint(importResp("00000000-0000-4000-8000-000000000000").Diagnostics); !strings.Contains(diags, "belongs to organization "+ssoSettingsTestOrgID) {
		t.Errorf("expected an organization mismatch error, got: %v", diags)
	}

	resp := importResp(ssoSettingsTestOrgID)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	want := SsoSettingsResourceModel{
		ID:                    types.StringValue(ssoSettingsTestOrgID),
		EnforceSso:            types.BoolValue(false),
		EnforceGoogleSso:      types.BoolValue(false),
		AdminSsoBypassEnabled: types.BoolValue(true),
		SsoMfaTrust:           types.StringValue("signal"),
	}
	if imported := ssoSettingsModel(t, resp.State); !reflect.DeepEqual(imported, want) {
		t.Errorf("expected the import to build the full state, got %+v", imported)
	}
}

func TestSsoSettingsValidateConfig(t *testing.T) {
	s := ssoSettingsTestSchema(t)
	r := &SsoSettingsResource{}

	validate := func(enforceSso, enforceGoogleSso types.Bool) string {
		model := SsoSettingsResourceModel{
			ID:                    types.StringNull(),
			EnforceSso:            enforceSso,
			EnforceGoogleSso:      enforceGoogleSso,
			AdminSsoBypassEnabled: types.BoolNull(),
			SsoMfaTrust:           types.StringNull(),
		}
		config := tfsdk.Config{Schema: s, Raw: ssoSettingsState(t, s, model).Raw}
		var resp resource.ValidateConfigResponse
		r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: config}, &resp)
		return fmt.Sprint(resp.Diagnostics)
	}

	if diags := validate(types.BoolValue(true), types.BoolValue(true)); !strings.Contains(diags, "cannot both be true") {
		t.Errorf("expected an error when both are true, got: %v", diags)
	}
	for _, c := range [][2]types.Bool{
		{types.BoolValue(true), types.BoolValue(false)},
		{types.BoolValue(true), types.BoolNull()},
		{types.BoolUnknown(), types.BoolValue(true)},
	} {
		if diags := validate(c[0], c[1]); diags != "[]" {
			t.Errorf("expected no error for %v, got: %v", c, diags)
		}
	}
}
