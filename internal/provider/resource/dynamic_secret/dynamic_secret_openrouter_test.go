package resource

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Fake backend that mimics the dynamic secret API for OpenRouter: reads and writes never return
// managementKey (DYNAMIC_SECRET_SECRET_FIELDS in providers/redact.ts), and PATCH merges the sent
// inputs into the stored ones ({...storedInputs, ...inputs} in dynamic-secret-service.ts), so an
// omitted field keeps its value and only an explicit null clears it.
type fakeOpenRouterDynamicSecretBackend struct {
	t           *testing.T
	secrets     map[string]map[string]any
	lastRequest map[string]any
}

func (b *fakeOpenRouterDynamicSecretBackend) writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		b.t.Error(err)
	}
}

func (b *fakeOpenRouterDynamicSecretBackend) respond(w http.ResponseWriter, name string) {
	inputs := map[string]any{}
	for key, value := range b.secrets[name] {
		if key != "managementKey" {
			inputs[key] = value
		}
	}
	b.writeJSON(w, http.StatusOK, map[string]any{"dynamicSecret": map[string]any{
		"id":         "ds-" + name,
		"name":       name,
		"type":       "openrouter",
		"defaultTTL": "1h",
		"inputs":     inputs,
	}})
}

func (b *fakeOpenRouterDynamicSecretBackend) decode(w http.ResponseWriter, r *http.Request) bool {
	b.lastRequest = map[string]any{}
	if err := json.NewDecoder(r.Body).Decode(&b.lastRequest); err != nil {
		b.writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
		return false
	}
	return true
}

func (b *fakeOpenRouterDynamicSecretBackend) handler() http.Handler {
	mux := http.NewServeMux()
	notFound := func(w http.ResponseWriter, name string) bool {
		if _, ok := b.secrets[name]; !ok {
			b.writeJSON(w, http.StatusNotFound, map[string]any{"message": "Dynamic secret not found"})
			return true
		}
		return false
	}

	mux.HandleFunc("POST /api/v1/dynamic-secrets", func(w http.ResponseWriter, r *http.Request) {
		if !b.decode(w, r) {
			return
		}
		provider, _ := b.lastRequest["provider"].(map[string]any)
		inputs, _ := provider["inputs"].(map[string]any)
		name, _ := b.lastRequest["name"].(string)
		b.secrets[name] = inputs
		b.respond(w, name)
	})

	mux.HandleFunc("GET /api/v1/dynamic-secrets/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if notFound(w, name) {
			return
		}
		b.respond(w, name)
	})

	mux.HandleFunc("PATCH /api/v1/dynamic-secrets/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if notFound(w, name) || !b.decode(w, r) {
			return
		}
		data, _ := b.lastRequest["data"].(map[string]any)
		inputs, _ := data["inputs"].(map[string]any)
		for key, value := range inputs {
			b.secrets[name][key] = value
		}
		b.respond(w, name)
	})

	mux.HandleFunc("DELETE /api/v1/dynamic-secrets/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if notFound(w, name) {
			return
		}
		b.respond(w, name)
		delete(b.secrets, name)
	})

	return mux
}

func newOpenRouterTestResource(t *testing.T) (*DynamicSecretBaseResource, *fakeOpenRouterDynamicSecretBackend, schema.Schema) {
	t.Helper()
	backend := &fakeOpenRouterDynamicSecretBackend{t: t, secrets: map[string]map[string]any{}}
	server := httptest.NewServer(backend.handler())
	t.Cleanup(server.Close)

	r, ok := NewDynamicSecretOpenRouterResource().(*DynamicSecretBaseResource)
	if !ok {
		t.Fatal("expected a *DynamicSecretBaseResource")
	}
	r.client = &infisical.Client{Config: infisical.Config{
		HttpClient:            resty.New().SetBaseURL(server.URL),
		IsMachineIdentityAuth: true,
	}}

	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics: %v", resp.Diagnostics)
	}
	return r, backend, resp.Schema
}

func openRouterConfiguration(t *testing.T, model DynamicSecretOpenRouterConfigurationModel) types.Object {
	t.Helper()
	obj, diags := types.ObjectValue(dynamicSecretOpenRouterConfigurationAttrTypes, map[string]attr.Value{
		"management_key":            model.ManagementKey,
		"management_key_wo":         model.ManagementKeyWO,
		"management_key_wo_version": model.ManagementKeyWOVersion,
		"workspace_id":              model.WorkspaceId,
		"limit":                     model.Limit,
		"limit_reset":               model.LimitReset,
		"include_byok_in_limit":     model.IncludeByokInLimit,
	})
	if diags.HasError() {
		t.Fatal(diags)
	}
	return obj
}

// Builds the plan and the config for a configuration: the plan never holds write-only values.
func openRouterPlanAndConfig(t *testing.T, s schema.Schema, id types.String, configuration DynamicSecretOpenRouterConfigurationModel) (tfsdk.Plan, tfsdk.Config) {
	t.Helper()
	model := DynamicSecretBaseResourceModel{
		ID:               id,
		Name:             types.StringValue("openrouter"),
		ProjectSlug:      types.StringValue("project"),
		EnvironmentSlug:  types.StringValue("dev"),
		Path:             types.StringValue("/"),
		DefaultTTL:       types.StringValue("1h"),
		MaxTTL:           types.StringNull(),
		UsernameTemplate: types.StringNull(),
		Configuration:    openRouterConfiguration(t, configuration),
	}
	rawConfig := tfsdk.Plan{Schema: s}
	if diags := rawConfig.Set(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	config := tfsdk.Config{Schema: s, Raw: rawConfig.Raw}

	configuration.ManagementKeyWO = types.StringNull()
	model.Configuration = openRouterConfiguration(t, configuration)
	plan := tfsdk.Plan{Schema: s}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return plan, config
}

func openRouterStateConfiguration(t *testing.T, state tfsdk.State) DynamicSecretOpenRouterConfigurationModel {
	t.Helper()
	var configuration DynamicSecretOpenRouterConfigurationModel
	if diags := state.GetAttribute(context.Background(), path.Root("configuration"), &configuration); diags.HasError() {
		t.Fatal(diags)
	}
	return configuration
}

func readOpenRouter(t *testing.T, r *DynamicSecretBaseResource, state tfsdk.State) tfsdk.State {
	t.Helper()
	resp := resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	return resp.State
}

func TestDynamicSecretOpenRouterLifecycle(t *testing.T) {
	ctx := context.Background()
	r, backend, s := newOpenRouterTestResource(t)

	full := DynamicSecretOpenRouterConfigurationModel{
		ManagementKey:          types.StringValue("sk-or-v1-management"),
		ManagementKeyWO:        types.StringNull(),
		ManagementKeyWOVersion: types.Int64Null(),
		WorkspaceId:            types.StringValue("ws-1"),
		Limit:                  types.StringValue("2.5"),
		LimitReset:             types.StringValue("monthly"),
		IncludeByokInLimit:     types.BoolValue(true),
	}
	plan, config := openRouterPlanAndConfig(t, s, types.StringUnknown(), full)
	createResp := resource.CreateResponse{State: tfsdk.State{Schema: s}}
	r.Create(ctx, resource.CreateRequest{Plan: plan, Config: config}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatal(createResp.Diagnostics)
	}

	provider, _ := backend.lastRequest["provider"].(map[string]any)
	if provider["type"] != "openrouter" {
		t.Errorf("expected provider type openrouter, got %v", provider["type"])
	}
	want := map[string]any{"managementKey": "sk-or-v1-management", "workspaceId": "ws-1", "limit": "2.5", "limitReset": "monthly", "includeByokInLimit": true}
	for key, value := range want {
		if got := backend.secrets["openrouter"][key]; got != value {
			t.Errorf("expected %s=%v to be sent on create, got %v", key, value, got)
		}
	}

	state := readOpenRouter(t, r, createResp.State)
	refreshed := openRouterStateConfiguration(t, state)
	if refreshed.ManagementKey.ValueString() != "sk-or-v1-management" {
		t.Errorf("expected the redacted management key to be kept from state, got %v", refreshed.ManagementKey)
	}
	if refreshed.WorkspaceId.ValueString() != "ws-1" || refreshed.Limit.ValueString() != "2.5" || refreshed.LimitReset.ValueString() != "monthly" || !refreshed.IncludeByokInLimit.ValueBool() {
		t.Errorf("unexpected configuration after read: %+v", refreshed)
	}

	cleared := full
	cleared.WorkspaceId = types.StringNull()
	cleared.Limit = types.StringNull()
	cleared.LimitReset = types.StringNull()
	cleared.IncludeByokInLimit = types.BoolValue(false)
	updatePlan, updateConfig := openRouterPlanAndConfig(t, s, types.StringValue("ds-openrouter"), cleared)
	updateResp := resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{Plan: updatePlan, Config: updateConfig, State: state}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatal(updateResp.Diagnostics)
	}
	for _, key := range []string{"workspaceId", "limit", "limitReset"} {
		if value, exists := backend.secrets["openrouter"][key]; !exists || value != nil {
			t.Errorf("expected %s to be cleared with an explicit null, got %v (present: %v)", key, value, exists)
		}
	}
	if got := backend.secrets["openrouter"]["includeByokInLimit"]; got != false {
		t.Errorf("expected includeByokInLimit=false to be sent, got %v", got)
	}

	refreshed = openRouterStateConfiguration(t, readOpenRouter(t, r, updateResp.State))
	if !refreshed.WorkspaceId.IsNull() || !refreshed.Limit.IsNull() || !refreshed.LimitReset.IsNull() || refreshed.IncludeByokInLimit.ValueBool() {
		t.Errorf("expected cleared fields after read, got %+v", refreshed)
	}
	if refreshed.ManagementKey.ValueString() != "sk-or-v1-management" {
		t.Errorf("expected the management key to survive the update, got %v", refreshed.ManagementKey)
	}

	deleteResp := resource.DeleteResponse{State: updateResp.State}
	r.Delete(ctx, resource.DeleteRequest{State: updateResp.State}, &deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatal(deleteResp.Diagnostics)
	}
	if _, exists := backend.secrets["openrouter"]; exists {
		t.Error("expected the dynamic secret to be deleted")
	}

	readResp := resource.ReadResponse{State: updateResp.State}
	r.Read(ctx, resource.ReadRequest{State: updateResp.State}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatal(readResp.Diagnostics)
	}
	if !readResp.State.Raw.IsNull() {
		t.Error("expected a 404 on read to remove the resource from state")
	}
}

func TestDynamicSecretOpenRouterWriteOnlyManagementKey(t *testing.T) {
	ctx := context.Background()
	r, backend, s := newOpenRouterTestResource(t)

	plan, config := openRouterPlanAndConfig(t, s, types.StringUnknown(), DynamicSecretOpenRouterConfigurationModel{
		ManagementKey:          types.StringNull(),
		ManagementKeyWO:        types.StringValue("sk-or-v1-write-only"),
		ManagementKeyWOVersion: types.Int64Value(1),
		WorkspaceId:            types.StringNull(),
		Limit:                  types.StringNull(),
		LimitReset:             types.StringNull(),
		IncludeByokInLimit:     types.BoolValue(false),
	})
	createResp := resource.CreateResponse{State: tfsdk.State{Schema: s}}
	r.Create(ctx, resource.CreateRequest{Plan: plan, Config: config}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatal(createResp.Diagnostics)
	}

	if got := backend.secrets["openrouter"]["managementKey"]; got != "sk-or-v1-write-only" {
		t.Errorf("expected the write-only management key to be sent, got %v", got)
	}

	configuration := openRouterStateConfiguration(t, readOpenRouter(t, r, createResp.State))
	if !configuration.ManagementKey.IsNull() || !configuration.ManagementKeyWO.IsNull() {
		t.Errorf("expected no management key in state, got %+v", configuration)
	}
	if configuration.ManagementKeyWOVersion.ValueInt64() != 1 {
		t.Errorf("expected management_key_wo_version to be kept, got %v", configuration.ManagementKeyWOVersion)
	}
}

// The write-only test above drives Create directly, so it relies on the schema to keep Terraform from
// putting management_key_wo into the plan or state.
func TestDynamicSecretOpenRouterManagementKeySchema(t *testing.T) {
	_, _, s := newOpenRouterTestResource(t)
	configuration, ok := s.Attributes["configuration"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatal("expected configuration to be a single nested attribute")
	}

	writeOnly, ok := configuration.Attributes["management_key_wo"].(schema.StringAttribute)
	if !ok || !writeOnly.WriteOnly || !writeOnly.Sensitive {
		t.Errorf("expected management_key_wo to be write-only and sensitive, got %+v", configuration.Attributes["management_key_wo"])
	}
	managementKey, ok := configuration.Attributes["management_key"].(schema.StringAttribute)
	if !ok || !managementKey.Sensitive {
		t.Errorf("expected management_key to be sensitive, got %+v", configuration.Attributes["management_key"])
	}
}
