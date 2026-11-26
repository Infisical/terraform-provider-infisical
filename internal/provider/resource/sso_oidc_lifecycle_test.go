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

const (
	ssoOidcTestOrgID    = "8a1f2b3c-4d5e-4f60-9a7b-8c9d0e1f2a3b"
	ssoOidcTestConfigID = "1e2d3c4b-5a69-4788-9a0b-1c2d3e4f5a6b"
)

type fakeOidcConfig struct {
	ID                     string   `json:"id"`
	OrgID                  string   `json:"orgId"`
	ConfigurationType      string   `json:"configurationType"`
	DiscoveryURL           string   `json:"discoveryURL"`
	Issuer                 string   `json:"issuer"`
	AuthorizationEndpoint  string   `json:"authorizationEndpoint"`
	TokenEndpoint          string   `json:"tokenEndpoint"`
	JwksUri                string   `json:"jwksUri"`
	UserinfoEndpoint       string   `json:"userinfoEndpoint"`
	ClientID               string   `json:"clientId,omitempty"`
	ClientSecret           string   `json:"clientSecret,omitempty"`
	AllowedEmailDomains    string   `json:"allowedEmailDomains"`
	JwtSignatureAlgorithm  string   `json:"jwtSignatureAlgorithm"`
	ManageGroupMemberships bool     `json:"manageGroupMemberships"`
	IsActive               bool     `json:"isActive"`
	MfaClaimName           string   `json:"mfaClaimName"`
	MfaAmrValues           []string `json:"mfaAmrValues"`
}

type fakeOidcBackend struct {
	t           *testing.T
	config      *fakeOidcConfig
	requests    []string
	bodies      []map[string]any
	patchBodies []map[string]any
}

func newFakeOidcBackend(t *testing.T) *fakeOidcBackend {
	return &fakeOidcBackend{t: t}
}

func (b *fakeOidcBackend) writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		b.t.Error(err)
	}
}

func (b *fakeOidcBackend) apiError(w http.ResponseWriter, status int, message string) {
	b.writeJSON(w, status, map[string]any{"statusCode": status, "message": message})
}

// sanitized is the POST, PATCH and DELETE response, which leaves out the client ID and secret.
func (b *fakeOidcBackend) sanitized() fakeOidcConfig {
	config := *b.config
	config.ClientID, config.ClientSecret = "", ""
	return config
}

func (b *fakeOidcBackend) handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/sso/oidc/config", func(w http.ResponseWriter, _ *http.Request) {
		if b.config == nil {
			b.apiError(w, http.StatusNotFound, "Failed to find OIDC SSO data")
			return
		}
		b.writeJSON(w, http.StatusOK, b.config)
	})

	mux.HandleFunc("POST /api/v1/sso/oidc/config", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			b.apiError(w, http.StatusUnprocessableEntity, "Invalid body")
			return
		}
		b.bodies = append(b.bodies, body)
		if b.config != nil {
			b.writeJSON(w, http.StatusConflict, map[string]any{"statusCode": http.StatusConflict, "message": fmt.Sprintf("An OIDC configuration already exists for organization with ID '%s'.", ssoOidcTestOrgID)})
			return
		}

		str := func(key string) string { value, _ := body[key].(string); return value }
		boolean := func(key string) bool { value, _ := body[key].(bool); return value }
		config := &fakeOidcConfig{
			ID:                     ssoOidcTestConfigID,
			OrgID:                  ssoOidcTestOrgID,
			ConfigurationType:      str("configurationType"),
			DiscoveryURL:           str("discoveryURL"),
			Issuer:                 str("issuer"),
			AuthorizationEndpoint:  str("authorizationEndpoint"),
			TokenEndpoint:          str("tokenEndpoint"),
			JwksUri:                str("jwksUri"),
			UserinfoEndpoint:       str("userinfoEndpoint"),
			ClientID:               str("clientId"),
			ClientSecret:           str("clientSecret"),
			AllowedEmailDomains:    joinFakeOidcDomains(str("allowedEmailDomains")),
			JwtSignatureAlgorithm:  str("jwtSignatureAlgorithm"),
			ManageGroupMemberships: boolean("manageGroupMemberships"),
			IsActive:               boolean("isActive"),
			MfaClaimName:           str("mfaClaimName"),
		}
		if config.MfaClaimName == "" {
			config.MfaClaimName = "amr"
		}
		if values, ok := body["mfaAmrValues"].([]any); ok {
			for _, value := range values {
				if s, ok := value.(string); ok {
					config.MfaAmrValues = append(config.MfaAmrValues, s)
				}
			}
		}
		b.config = config
		b.writeJSON(w, http.StatusOK, b.sanitized())
	})

	mux.HandleFunc("PATCH /api/v1/sso/oidc/config", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			b.apiError(w, http.StatusUnprocessableEntity, "Invalid body")
			return
		}
		b.patchBodies = append(b.patchBodies, body)
		if b.config == nil {
			b.apiError(w, http.StatusNotFound, fmt.Sprintf("OIDC configuration for organization with ID '%s' not found", ssoOidcTestOrgID))
			return
		}

		stringFields := map[string]*string{
			"configurationType":     &b.config.ConfigurationType,
			"discoveryURL":          &b.config.DiscoveryURL,
			"issuer":                &b.config.Issuer,
			"authorizationEndpoint": &b.config.AuthorizationEndpoint,
			"tokenEndpoint":         &b.config.TokenEndpoint,
			"jwksUri":               &b.config.JwksUri,
			"userinfoEndpoint":      &b.config.UserinfoEndpoint,
			"clientId":              &b.config.ClientID,
			"clientSecret":          &b.config.ClientSecret,
			"jwtSignatureAlgorithm": &b.config.JwtSignatureAlgorithm,
			"mfaClaimName":          &b.config.MfaClaimName,
		}
		boolFields := map[string]*bool{
			"manageGroupMemberships": &b.config.ManageGroupMemberships,
			"isActive":               &b.config.IsActive,
		}
		for key, value := range body {
			switch {
			case stringFields[key] != nil:
				*stringFields[key], _ = value.(string)
			case boolFields[key] != nil:
				*boolFields[key], _ = value.(bool)
			case key == "allowedEmailDomains":
				domains, _ := value.(string)
				b.config.AllowedEmailDomains = joinFakeOidcDomains(domains)
			case key == "mfaAmrValues":
				b.config.MfaAmrValues = nil
				values, _ := value.([]any)
				for _, v := range values {
					if s, ok := v.(string); ok {
						b.config.MfaAmrValues = append(b.config.MfaAmrValues, s)
					}
				}
			}
		}
		b.writeJSON(w, http.StatusOK, b.sanitized())
	})

	mux.HandleFunc("DELETE /api/v1/sso/oidc/config", func(w http.ResponseWriter, _ *http.Request) {
		if b.config == nil {
			b.apiError(w, http.StatusNotFound, fmt.Sprintf("OIDC configuration for organization with ID '%s' not found", ssoOidcTestOrgID))
			return
		}
		deleted := b.sanitized()
		b.config = nil
		b.writeJSON(w, http.StatusOK, deleted)
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b.requests = append(b.requests, r.Method+" "+r.URL.Path)
		mux.ServeHTTP(w, r)
	})
}

// joinFakeOidcDomains copies the route schema, which trims each domain and joins them with ", ".
func joinFakeOidcDomains(value string) string {
	return strings.Join(splitOidcAllowedEmailDomains(value), ", ")
}

func newSsoOidcTestResource(t *testing.T, backend *fakeOidcBackend) *SsoOidcResource {
	t.Helper()
	server := httptest.NewServer(backend.handler())
	t.Cleanup(server.Close)
	return &SsoOidcResource{client: &infisical.Client{Config: infisical.Config{
		HttpClient:            resty.New().SetBaseURL(server.URL),
		IsMachineIdentityAuth: true,
	}}}
}

func ssoOidcTestSchema(t *testing.T) schema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	NewSsoOidcResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func ssoOidcStringSet(t *testing.T, values ...string) types.Set {
	t.Helper()
	set, diags := types.SetValueFrom(context.Background(), types.StringType, values)
	if diags.HasError() {
		t.Fatal(diags)
	}
	return set
}

// ssoOidcBaseModel is a discovery URL configuration with client_secret and the schema defaults.
func ssoOidcBaseModel() SsoOidcResourceModel {
	return SsoOidcResourceModel{
		ID:                     types.StringUnknown(),
		DiscoveryURL:           types.StringValue("https://idp.example.com/.well-known/openid-configuration"),
		ClientID:               types.StringValue("client-id"),
		ClientSecret:           types.StringValue("client-secret"),
		ClientSecretWO:         types.StringNull(),
		ClientSecretWOVersion:  types.Int64Null(),
		AllowedEmailDomains:    types.SetNull(types.StringType),
		JwtSignatureAlgorithm:  types.StringValue("RS256"),
		ManageGroupMemberships: types.BoolValue(false),
		IsActive:               types.BoolValue(false),
		MfaClaimName:           types.StringValue("amr"),
		MfaAmrValues:           types.SetNull(types.StringType),
	}
}

// ssoOidcPlanAndConfig returns the plan and the config. The plan never holds client_secret_wo; the config does.
func ssoOidcPlanAndConfig(t *testing.T, s schema.Schema, model SsoOidcResourceModel) (tfsdk.Plan, tfsdk.Config) {
	t.Helper()
	ctx := context.Background()

	config := tfsdk.Config{Schema: s}
	configModel := model
	configModel.ID = types.StringNull()
	configState := tfsdk.State{Schema: s}
	if diags := configState.Set(ctx, &configModel); diags.HasError() {
		t.Fatal(diags)
	}
	config.Raw = configState.Raw

	planModel := model
	planModel.ClientSecretWO = types.StringNull()
	plan := tfsdk.Plan{Schema: s}
	if diags := plan.Set(ctx, &planModel); diags.HasError() {
		t.Fatal(diags)
	}
	return plan, config
}

func ssoOidcState(t *testing.T, s schema.Schema, model SsoOidcResourceModel) tfsdk.State {
	t.Helper()
	state := tfsdk.State{Schema: s}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return state
}

func ssoOidcModel(t *testing.T, state tfsdk.State) SsoOidcResourceModel {
	t.Helper()
	var model SsoOidcResourceModel
	if diags := state.Get(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return model
}

func createSsoOidc(r *SsoOidcResource, plan tfsdk.Plan, config tfsdk.Config) resource.CreateResponse {
	resp := resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema}}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan, Config: config}, &resp)
	return resp
}

func mustCreateSsoOidc(t *testing.T, r *SsoOidcResource, s schema.Schema, model SsoOidcResourceModel) tfsdk.State {
	t.Helper()
	plan, config := ssoOidcPlanAndConfig(t, s, model)
	resp := createSsoOidc(r, plan, config)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	return resp.State
}

func updateSsoOidc(r *SsoOidcResource, state tfsdk.State, plan tfsdk.Plan, config tfsdk.Config) resource.UpdateResponse {
	resp := resource.UpdateResponse{State: state}
	r.Update(context.Background(), resource.UpdateRequest{State: state, Plan: plan, Config: config}, &resp)
	return resp
}

func mustUpdateSsoOidc(t *testing.T, r *SsoOidcResource, state tfsdk.State, model SsoOidcResourceModel) SsoOidcResourceModel {
	t.Helper()
	model.ID = ssoOidcModel(t, state).ID
	plan, config := ssoOidcPlanAndConfig(t, ssoOidcTestSchema(t), model)
	resp := updateSsoOidc(r, state, plan, config)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	return ssoOidcModel(t, resp.State)
}

func readSsoOidc(r *SsoOidcResource, state tfsdk.State) resource.ReadResponse {
	resp := resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
	return resp
}

func TestSsoOidcLifecycle(t *testing.T) {
	s := ssoOidcTestSchema(t)
	backend := newFakeOidcBackend(t)
	r := newSsoOidcTestResource(t, backend)

	model := ssoOidcBaseModel()
	model.AllowedEmailDomains = ssoOidcStringSet(t, "example.com")
	model.IsActive = types.BoolValue(true)
	state := mustCreateSsoOidc(t, r, s, model)

	want := map[string]any{
		"configurationType":      "discoveryURL",
		"discoveryURL":           "https://idp.example.com/.well-known/openid-configuration",
		"clientId":               "client-id",
		"clientSecret":           "client-secret",
		"allowedEmailDomains":    "example.com",
		"jwtSignatureAlgorithm":  "RS256",
		"manageGroupMemberships": false,
		"isActive":               true,
		"mfaClaimName":           "amr",
	}
	if !reflect.DeepEqual(backend.bodies[0], want) {
		t.Errorf("unexpected create body:\n got: %v\nwant: %v", backend.bodies[0], want)
	}

	created := ssoOidcModel(t, state)
	if created.ID.ValueString() != ssoOidcTestConfigID || created.Custom != nil || created.ClientSecret.ValueString() != "client-secret" ||
		!created.AllowedEmailDomains.Equal(ssoOidcStringSet(t, "example.com")) || !created.MfaAmrValues.IsNull() || !created.IsActive.ValueBool() {
		t.Errorf("unexpected state after create: %+v", created)
	}

	readResp := readSsoOidc(r, state)
	if readResp.Diagnostics.HasError() {
		t.Fatal(readResp.Diagnostics)
	}
	if refreshed := ssoOidcModel(t, readResp.State); !reflect.DeepEqual(refreshed, created) {
		t.Errorf("expected no drift after a refresh, got %+v, want %+v", refreshed, created)
	}

	var deleteResp resource.DeleteResponse
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, &deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatal(deleteResp.Diagnostics)
	}
	if backend.config != nil {
		t.Error("expected delete to remove the config")
	}
}

// A custom configuration with client_secret_wo sends the secret from the config and keeps it out of state.
func TestSsoOidcCustomWithWriteOnlySecret(t *testing.T) {
	s := ssoOidcTestSchema(t)
	backend := newFakeOidcBackend(t)
	r := newSsoOidcTestResource(t, backend)

	model := ssoOidcBaseModel()
	model.DiscoveryURL = types.StringNull()
	model.Custom = &SsoOidcCustomModel{
		Issuer:                types.StringValue("https://idp.example.com"),
		AuthorizationEndpoint: types.StringValue("https://idp.example.com/authorize"),
		TokenEndpoint:         types.StringValue("https://idp.example.com/token"),
		JwksUri:               types.StringValue("https://idp.example.com/jwks"),
		UserinfoEndpoint:      types.StringValue("https://idp.example.com/userinfo"),
	}
	model.ClientSecret = types.StringNull()
	model.ClientSecretWO = types.StringValue("write-only-secret")
	model.ClientSecretWOVersion = types.Int64Value(1)
	model.MfaAmrValues = ssoOidcStringSet(t, "otp", "hwk")
	state := mustCreateSsoOidc(t, r, s, model)

	body := backend.bodies[0]
	if body["configurationType"] != "custom" || body["clientSecret"] != "write-only-secret" || body["tokenEndpoint"] != "https://idp.example.com/token" || body["discoveryURL"] != nil {
		t.Errorf("unexpected create body: %v", body)
	}

	created := ssoOidcModel(t, state)
	if !created.ClientSecret.IsNull() || !created.ClientSecretWO.IsNull() || created.ClientSecretWOVersion.ValueInt64() != 1 {
		t.Errorf("expected no secret in state and the version kept, got %+v", created)
	}
	if created.Custom == nil || created.Custom.JwksUri.ValueString() != "https://idp.example.com/jwks" || !created.DiscoveryURL.IsNull() {
		t.Errorf("unexpected endpoints in state: %+v", created)
	}
	if !created.MfaAmrValues.Equal(ssoOidcStringSet(t, "otp", "hwk")) {
		t.Errorf("unexpected mfa_amr_values: %v", created.MfaAmrValues)
	}

	backend.config.DiscoveryURL = "https://stale.example.com"
	if refreshed := ssoOidcModel(t, readSsoOidc(r, state).State); !refreshed.DiscoveryURL.IsNull() {
		t.Errorf("expected discovery_url to stay null in custom mode, got %v", refreshed.DiscoveryURL)
	}
}

func TestSsoOidcCreateAlreadyExists(t *testing.T) {
	s := ssoOidcTestSchema(t)
	backend := newFakeOidcBackend(t)
	r := newSsoOidcTestResource(t, backend)
	mustCreateSsoOidc(t, r, s, ssoOidcBaseModel())

	plan, config := ssoOidcPlanAndConfig(t, s, ssoOidcBaseModel())
	resp := createSsoOidc(r, plan, config)
	if !resp.Diagnostics.HasError() || !strings.Contains(fmt.Sprint(resp.Diagnostics), "terraform import") {
		t.Errorf("expected an error that points to terraform import, got: %v", resp.Diagnostics)
	}
}

// The API joins allowedEmailDomains with ", " and the state keeps the domains as a set.
func TestSsoOidcReadAllowedEmailDomains(t *testing.T) {
	s := ssoOidcTestSchema(t)
	backend := newFakeOidcBackend(t)
	r := newSsoOidcTestResource(t, backend)
	state := mustCreateSsoOidc(t, r, s, ssoOidcBaseModel())

	backend.config.AllowedEmailDomains = "example.com, example.org"
	refreshed := ssoOidcModel(t, readSsoOidc(r, state).State)
	if !refreshed.AllowedEmailDomains.Equal(ssoOidcStringSet(t, "example.com", "example.org")) {
		t.Errorf("unexpected allowed_email_domains: %v", refreshed.AllowedEmailDomains)
	}
}

func TestSsoOidcReadDropsDeletedConfig(t *testing.T) {
	s := ssoOidcTestSchema(t)
	backend := newFakeOidcBackend(t)
	r := newSsoOidcTestResource(t, backend)
	state := mustCreateSsoOidc(t, r, s, ssoOidcBaseModel())

	backend.config = nil
	readResp := readSsoOidc(r, state)
	if readResp.Diagnostics.HasError() {
		t.Fatal(readResp.Diagnostics)
	}
	if !readResp.State.Raw.IsNull() {
		t.Error("expected the resource to be removed from state")
	}
}

func TestSsoOidcReadEmptyID(t *testing.T) {
	backend := newFakeOidcBackend(t)
	r := newSsoOidcTestResource(t, backend)

	model := ssoOidcBaseModel()
	model.ID = types.StringValue("")
	if readResp := readSsoOidc(r, ssoOidcState(t, ssoOidcTestSchema(t), model)); !readResp.State.Raw.IsNull() {
		t.Error("expected the resource to be removed from state")
	}
	if len(backend.requests) != 0 {
		t.Errorf("expected no API call, got %v", backend.requests)
	}
}

func TestSsoOidcDeleteNotFound(t *testing.T) {
	s := ssoOidcTestSchema(t)
	backend := newFakeOidcBackend(t)
	r := newSsoOidcTestResource(t, backend)

	model := ssoOidcBaseModel()
	model.ID = types.StringValue(ssoOidcTestConfigID)
	var resp resource.DeleteResponse
	r.Delete(context.Background(), resource.DeleteRequest{State: ssoOidcState(t, s, model)}, &resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("expected a 404 on delete to count as success, got: %v", resp.Diagnostics)
	}
}

func TestSsoOidcImport(t *testing.T) {
	ctx := context.Background()
	s := ssoOidcTestSchema(t)
	backend := newFakeOidcBackend(t)
	r := newSsoOidcTestResource(t, backend)

	emptyState := func() tfsdk.State { return ssoOidcState(t, s, ssoOidcBaseModel()) }
	importErr := func(id string) string {
		resp := resource.ImportStateResponse{State: emptyState()}
		r.ImportState(ctx, resource.ImportStateRequest{ID: id}, &resp)
		return fmt.Sprint(resp.Diagnostics)
	}

	if diags := importErr("not-a-uuid"); !strings.Contains(diags, "Invalid import ID") {
		t.Errorf("expected an Invalid import ID error, got: %v", diags)
	}
	if diags := importErr(ssoOidcTestOrgID); !strings.Contains(diags, "has no OIDC configuration") {
		t.Errorf("expected a missing configuration error, got: %v", diags)
	}

	model := ssoOidcBaseModel()
	model.AllowedEmailDomains = ssoOidcStringSet(t, "example.com")
	created := ssoOidcModel(t, mustCreateSsoOidc(t, r, s, model))

	if diags := importErr("00000000-0000-4000-8000-000000000000"); !strings.Contains(diags, "belongs to organization "+ssoOidcTestOrgID) {
		t.Errorf("expected an organization mismatch error, got: %v", diags)
	}

	resp := resource.ImportStateResponse{State: emptyState()}
	r.ImportState(ctx, resource.ImportStateRequest{ID: ssoOidcTestOrgID}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	readResp := readSsoOidc(r, resp.State)
	if readResp.Diagnostics.HasError() {
		t.Fatal(readResp.Diagnostics)
	}
	if imported := ssoOidcModel(t, readResp.State); !reflect.DeepEqual(imported, created) {
		t.Errorf("expected the import to rebuild the full state, got %+v, want %+v", imported, created)
	}
}

func TestSsoOidcUpdateSendsFullConfig(t *testing.T) {
	s := ssoOidcTestSchema(t)
	backend := newFakeOidcBackend(t)
	r := newSsoOidcTestResource(t, backend)

	model := ssoOidcBaseModel()
	model.AllowedEmailDomains = ssoOidcStringSet(t, "example.com", "example.org")
	model.IsActive = types.BoolValue(true)
	model.MfaAmrValues = ssoOidcStringSet(t, "otp")
	state := mustCreateSsoOidc(t, r, s, model)

	model.ClientID = types.StringValue("new-client-id")
	model.AllowedEmailDomains = types.SetNull(types.StringType)
	model.IsActive = types.BoolValue(false)
	model.JwtSignatureAlgorithm = types.StringValue("RS512")
	model.MfaAmrValues = types.SetNull(types.StringType)
	updated := mustUpdateSsoOidc(t, r, state, model)

	want := map[string]any{
		"configurationType":      "discoveryURL",
		"discoveryURL":           "https://idp.example.com/.well-known/openid-configuration",
		"clientId":               "new-client-id",
		"clientSecret":           "client-secret",
		"allowedEmailDomains":    "",
		"jwtSignatureAlgorithm":  "RS512",
		"manageGroupMemberships": false,
		"isActive":               false,
		"mfaClaimName":           "amr",
		"mfaAmrValues":           []any{},
	}
	if len(backend.patchBodies) != 1 || !reflect.DeepEqual(backend.patchBodies[0], want) {
		t.Fatalf("unexpected update bodies:\n got: %v\nwant: [%v]", backend.patchBodies, want)
	}
	// The update response has no client ID or secret, so state takes them from the plan.
	if updated.ClientID.ValueString() != "new-client-id" || updated.ClientSecret.ValueString() != "client-secret" ||
		!updated.AllowedEmailDomains.IsNull() || !updated.MfaAmrValues.IsNull() || updated.IsActive.ValueBool() || updated.JwtSignatureAlgorithm.ValueString() != "RS512" {
		t.Errorf("unexpected state after update: %+v", updated)
	}
	if refreshed := ssoOidcModel(t, readSsoOidc(r, ssoOidcState(t, s, updated)).State); !reflect.DeepEqual(refreshed, updated) {
		t.Errorf("expected no drift after a refresh, got %+v, want %+v", refreshed, updated)
	}
}

// Moving from client_secret to client_secret_wo removes the secret from state. A new client_secret_wo with a new
// version sends the secret from the config, because the plan never holds it.
func TestSsoOidcUpdateWriteOnlySecret(t *testing.T) {
	s := ssoOidcTestSchema(t)
	backend := newFakeOidcBackend(t)
	r := newSsoOidcTestResource(t, backend)
	state := mustCreateSsoOidc(t, r, s, ssoOidcBaseModel())

	model := ssoOidcBaseModel()
	model.ClientSecret = types.StringNull()
	model.ClientSecretWO = types.StringValue("client-secret")
	model.ClientSecretWOVersion = types.Int64Value(1)
	updated := mustUpdateSsoOidc(t, r, state, model)
	if !updated.ClientSecret.IsNull() || updated.ClientSecretWOVersion.ValueInt64() != 1 {
		t.Errorf("expected no secret in state and version 1, got %+v", updated)
	}

	model.ClientSecretWO = types.StringValue("rotated-secret")
	model.ClientSecretWOVersion = types.Int64Value(2)
	updated = mustUpdateSsoOidc(t, r, ssoOidcState(t, s, updated), model)
	if got := backend.patchBodies[1]["clientSecret"]; got != "rotated-secret" {
		t.Errorf("expected the new secret to be sent, got %v", got)
	}
	if !updated.ClientSecret.IsNull() || updated.ClientSecretWOVersion.ValueInt64() != 2 {
		t.Errorf("expected no secret in state and version 2, got %+v", updated)
	}
}

// Moving from discovery_url to custom sends the configuration type and the custom endpoints, not the discovery URL.
func TestSsoOidcUpdateModeSwitch(t *testing.T) {
	s := ssoOidcTestSchema(t)
	backend := newFakeOidcBackend(t)
	r := newSsoOidcTestResource(t, backend)
	state := mustCreateSsoOidc(t, r, s, ssoOidcBaseModel())

	model := ssoOidcBaseModel()
	model.DiscoveryURL = types.StringNull()
	model.Custom = &SsoOidcCustomModel{
		Issuer:                types.StringValue("https://idp.example.com"),
		AuthorizationEndpoint: types.StringValue("https://idp.example.com/authorize"),
		TokenEndpoint:         types.StringValue("https://idp.example.com/token"),
		JwksUri:               types.StringValue("https://idp.example.com/jwks"),
		UserinfoEndpoint:      types.StringValue("https://idp.example.com/userinfo"),
	}
	updated := mustUpdateSsoOidc(t, r, state, model)

	body := backend.patchBodies[0]
	if body["configurationType"] != "custom" || body["issuer"] != "https://idp.example.com" || body["userinfoEndpoint"] != "https://idp.example.com/userinfo" {
		t.Errorf("unexpected update body: %v", body)
	}
	if _, ok := body["discoveryURL"]; ok {
		t.Errorf("expected no discoveryURL in custom mode, got %v", body)
	}
	if updated.Custom == nil || !updated.DiscoveryURL.IsNull() || updated.Custom.Issuer.ValueString() != "https://idp.example.com" {
		t.Errorf("unexpected state after the mode switch: %+v", updated)
	}

	// Back to discovery_url: the custom endpoints are left out.
	mustUpdateSsoOidc(t, r, ssoOidcState(t, s, updated), ssoOidcBaseModel())
	body = backend.patchBodies[1]
	if body["configurationType"] != "discoveryURL" || body["discoveryURL"] != "https://idp.example.com/.well-known/openid-configuration" {
		t.Errorf("unexpected body when switching back: %v", body)
	}
	if _, ok := body["issuer"]; ok {
		t.Errorf("expected no custom endpoints in discovery URL mode, got %v", body)
	}
}

func TestSsoOidcUpdateDeletedConfig(t *testing.T) {
	s := ssoOidcTestSchema(t)
	backend := newFakeOidcBackend(t)
	r := newSsoOidcTestResource(t, backend)
	state := mustCreateSsoOidc(t, r, s, ssoOidcBaseModel())
	backend.config = nil

	model := ssoOidcBaseModel()
	model.ID = types.StringValue(ssoOidcTestConfigID)
	model.IsActive = types.BoolValue(true)
	plan, config := ssoOidcPlanAndConfig(t, s, model)
	resp := updateSsoOidc(r, state, plan, config)
	if !resp.Diagnostics.HasError() || !strings.Contains(fmt.Sprint(resp.Diagnostics), "no longer exists") {
		t.Errorf("expected a missing configuration error, got: %v", resp.Diagnostics)
	}
}
