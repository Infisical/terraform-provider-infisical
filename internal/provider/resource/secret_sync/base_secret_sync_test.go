package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// fakeSyncStep is one status the sync job reports. Each GET applies the next step, which stands in for the
// background job moving forward between two polls.
type fakeSyncStep struct {
	status  infisical.SecretSyncStatus
	message string
	deleted bool // the sync is deleted out of band at this point
}

type fakeSecretSync struct {
	data  infisical.SecretSync
	steps []fakeSyncStep
}

// fakeSecretSyncBackend answers like the secret sync routes in Infisical core. Create and update set syncStatus to
// "pending" and queue a job only when auto sync is on, and an update that omits isAutoSyncEnabled keeps the current
// value (createSecretSync and updateSecretSync in secret-sync-service.ts). The queued job then reports
// running, succeeded or failed (secret-sync-queue.ts).
type fakeSecretSyncBackend struct {
	t  *testing.T
	mu sync.Mutex

	syncs map[string]*fakeSecretSync
	// nextRun is the status script for the job that the next create or update queues.
	nextRun []fakeSyncStep

	gets          int
	patches       int
	createBodies  []map[string]any
	patchBodies   []map[string]any
	statusAtPatch []string
}

func newFakeSecretSyncBackend(t *testing.T) *fakeSecretSyncBackend {
	return &fakeSecretSyncBackend{t: t, syncs: map[string]*fakeSecretSync{}}
}

func (b *fakeSecretSyncBackend) writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		b.t.Error(err)
	}
}

func (b *fakeSecretSyncBackend) notFound(w http.ResponseWriter, id string) {
	b.writeJSON(w, http.StatusNotFound, map[string]any{"statusCode": http.StatusNotFound, "message": fmt.Sprintf("Could not find Secret Sync with ID %q", id)})
}

func decodeFakeBody[T any](t *testing.T, r *http.Request) (map[string]any, T) {
	t.Helper()
	var typed T
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &typed); err != nil {
		t.Fatal(err)
	}
	return generic, typed
}

func (b *fakeSecretSyncBackend) queueRun(s *fakeSecretSync) {
	pending := infisical.SecretSyncStatusPending
	s.data.SyncStatus = &pending
	s.steps = append([]fakeSyncStep(nil), b.nextRun...)
}

func (b *fakeSecretSyncBackend) handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/secret-syncs/{app}", func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		defer b.mu.Unlock()

		body, req := decodeFakeBody[infisical.CreateSecretSyncRequest](b.t, r)
		b.createBodies = append(b.createBodies, body)

		s := &fakeSecretSync{data: infisical.SecretSync{
			ID:                uuid.NewString(),
			Name:              req.Name,
			Description:       req.Description,
			AutoSyncEnabled:   req.AutoSyncEnabled,
			ProjectID:         req.ProjectID,
			ConnectionID:      req.ConnectionID,
			Connection:        infisical.SecretSyncConnection{ConnectionID: req.ConnectionID},
			Environment:       infisical.SecretSyncEnvironment{Slug: req.Environment},
			SecretFolder:      infisical.SecretSyncFolder{Path: req.SecretPath},
			SyncOptions:       req.SyncOptions,
			DestinationConfig: req.DestinationConfig,
		}}
		if s.data.AutoSyncEnabled {
			b.queueRun(s)
		}
		b.syncs[s.data.ID] = s
		b.writeJSON(w, http.StatusOK, infisical.CreateSecretSyncResponse{SecretSync: s.data})
	})

	mux.HandleFunc("GET /api/v1/secret-syncs/{app}/{id}", func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		defer b.mu.Unlock()

		b.gets++
		id := r.PathValue("id")
		s, ok := b.syncs[id]
		if !ok {
			b.notFound(w, id)
			return
		}
		if len(s.steps) > 0 {
			step := s.steps[0]
			s.steps = s.steps[1:]
			if step.deleted {
				delete(b.syncs, id)
				b.notFound(w, id)
				return
			}
			status := step.status
			s.data.SyncStatus = &status
			if step.message != "" {
				message := step.message
				s.data.LastSyncMessage = &message
			}
		}
		b.writeJSON(w, http.StatusOK, infisical.GetSecretSyncByIdResponse{SecretSync: s.data})
	})

	mux.HandleFunc("PATCH /api/v1/secret-syncs/{app}/{id}", func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		defer b.mu.Unlock()

		id := r.PathValue("id")
		s, ok := b.syncs[id]
		if !ok {
			b.notFound(w, id)
			return
		}

		body, req := decodeFakeBody[infisical.UpdateSecretSyncRequest](b.t, r)
		b.patches++
		b.patchBodies = append(b.patchBodies, body)
		b.statusAtPatch = append(b.statusAtPatch, syncStatusString(s.data))

		s.data.Name = req.Name
		s.data.Description = req.Description
		s.data.SyncOptions = req.SyncOptions
		s.data.DestinationConfig = req.DestinationConfig
		if value, sent := body["isAutoSyncEnabled"].(bool); sent {
			s.data.AutoSyncEnabled = value
		}
		if s.data.AutoSyncEnabled {
			b.queueRun(s)
		}
		b.writeJSON(w, http.StatusOK, infisical.UpdateSecretSyncResponse{SecretSync: s.data})
	})

	mux.HandleFunc("DELETE /api/v1/secret-syncs/{app}/{id}", func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		defer b.mu.Unlock()

		id := r.PathValue("id")
		s, ok := b.syncs[id]
		if !ok {
			b.notFound(w, id)
			return
		}
		delete(b.syncs, id)
		b.writeJSON(w, http.StatusOK, infisical.DeleteSecretSyncResponse{SecretSync: s.data})
	})

	return mux
}

// setStatus sets the current status of a sync and the steps its job reports next.
func (b *fakeSecretSyncBackend) setStatus(id string, status infisical.SecretSyncStatus, steps ...fakeSyncStep) {
	b.mu.Lock()
	defer b.mu.Unlock()
	s := b.syncs[id]
	s.data.SyncStatus = &status
	s.steps = steps
}

type secretSyncTestVariant struct {
	name              string
	newResource       func() resource.Resource
	syncOptions       map[string]attr.Value
	destinationConfig map[string]attr.Value
}

// One variant that can't import secrets and one that can, because the base builds different docs for them.
var secretSyncTestVariants = []secretSyncTestVariant{
	{
		name:        "azure devops",
		newResource: NewSecretSyncAzureDevOpsResource,
		syncOptions: map[string]attr.Value{
			"disable_secret_deletion": types.BoolValue(false),
			"key_schema":              types.StringNull(),
		},
		destinationConfig: map[string]attr.Value{"devops_project_id": types.StringValue("devops-project")},
	},
	{
		name:        "azure key vault",
		newResource: NewSecretSyncAzureKeyVaultResource,
		syncOptions: map[string]attr.Value{
			"initial_sync_behavior":   types.StringValue("import-prioritize-source"),
			"disable_secret_deletion": types.BoolValue(false),
			"key_schema":              types.StringNull(),
		},
		destinationConfig: map[string]attr.Value{"vault_base_url": types.StringValue("https://vault.example.com")},
	},
}

func forEachSecretSyncVariant(t *testing.T, test func(t *testing.T, v secretSyncTestVariant)) {
	for _, v := range secretSyncTestVariants {
		t.Run(v.name, func(t *testing.T) { test(t, v) })
	}
}

type secretSyncTestHarness struct {
	t       *testing.T
	r       *SecretSyncBaseResource
	schema  schema.Schema
	backend *fakeSecretSyncBackend
	variant secretSyncTestVariant
}

func newSecretSyncTestHarness(t *testing.T, v secretSyncTestVariant) *secretSyncTestHarness {
	t.Helper()
	backend := newFakeSecretSyncBackend(t)
	server := httptest.NewServer(backend.handler())
	t.Cleanup(server.Close)

	r, ok := v.newResource().(*SecretSyncBaseResource)
	if !ok {
		t.Fatalf("expected %s to be a *SecretSyncBaseResource", v.name)
	}
	r.client = &infisical.Client{Config: infisical.Config{
		HttpClient:            resty.New().SetBaseURL(server.URL),
		IsMachineIdentityAuth: true,
	}}
	r.pollInterval = time.Millisecond

	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics: %v", schemaResp.Diagnostics)
	}

	return &secretSyncTestHarness{t: t, r: r, schema: schemaResp.Schema, backend: backend, variant: v}
}

func (h *secretSyncTestHarness) object(attribute string, values map[string]attr.Value) types.Object {
	h.t.Helper()
	objectType, ok := h.schema.Attributes[attribute].GetType().(types.ObjectType)
	if !ok {
		h.t.Fatalf("expected %s to be an object", attribute)
	}
	value, diags := types.ObjectValue(objectType.AttrTypes, values)
	if diags.HasError() {
		h.t.Fatal(diags)
	}
	return value
}

func secretSyncTimeouts(values map[string]string) timeouts.Value {
	attrTypes := map[string]attr.Type{"create": types.StringType, "update": types.StringType}
	if values == nil {
		return timeouts.Value{Object: types.ObjectNull(attrTypes)}
	}
	attrValues := map[string]attr.Value{"create": types.StringNull(), "update": types.StringNull()}
	for key, value := range values {
		attrValues[key] = types.StringValue(value)
	}
	return timeouts.Value{Object: types.ObjectValueMust(attrTypes, attrValues)}
}

// model is what the plan looks like on create, with the given wait settings.
func (h *secretSyncTestHarness) model(wait bool, timeoutValues map[string]string) SecretSyncBaseResourceModel {
	return SecretSyncBaseResourceModel{
		ID:                    types.StringUnknown(),
		ConnectionID:          types.StringValue("connection-id"),
		Name:                  types.StringValue("my-sync"),
		ProjectID:             types.StringValue("project-id"),
		Description:           types.StringNull(),
		Environment:           types.StringValue("prod"),
		SecretPath:            types.StringValue("/"),
		SyncOptions:           h.object("sync_options", h.variant.syncOptions),
		AutoSyncEnabled:       types.BoolValue(true),
		DestinationConfig:     h.object("destination_config", h.variant.destinationConfig),
		WaitForSyncCompletion: types.BoolValue(wait),
		Timeouts:              secretSyncTimeouts(timeoutValues),
	}
}

func (h *secretSyncTestHarness) plan(model SecretSyncBaseResourceModel) tfsdk.Plan {
	h.t.Helper()
	plan := tfsdk.Plan{Schema: h.schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		h.t.Fatal(diags)
	}
	return plan
}

func (h *secretSyncTestHarness) state(model SecretSyncBaseResourceModel) tfsdk.State {
	h.t.Helper()
	state := tfsdk.State{Schema: h.schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		h.t.Fatal(diags)
	}
	return state
}

func (h *secretSyncTestHarness) stateModel(state tfsdk.State) SecretSyncBaseResourceModel {
	h.t.Helper()
	var model SecretSyncBaseResourceModel
	if diags := state.Get(context.Background(), &model); diags.HasError() {
		h.t.Fatal(diags)
	}
	return model
}

func (h *secretSyncTestHarness) create(model SecretSyncBaseResourceModel) resource.CreateResponse {
	resp := resource.CreateResponse{State: tfsdk.State{Schema: h.schema}}
	h.r.Create(context.Background(), resource.CreateRequest{Plan: h.plan(model)}, &resp)
	return resp
}

// applied creates a sync without waiting and returns the state model, for tests that start from an applied state.
func (h *secretSyncTestHarness) applied() SecretSyncBaseResourceModel {
	h.t.Helper()
	resp := h.create(h.model(false, nil))
	if resp.Diagnostics.HasError() {
		h.t.Fatal(resp.Diagnostics)
	}
	return h.stateModel(resp.State)
}

func (h *secretSyncTestHarness) update(state, plan SecretSyncBaseResourceModel) resource.UpdateResponse {
	stateValue := h.state(state)
	resp := resource.UpdateResponse{State: stateValue}
	h.r.Update(context.Background(), resource.UpdateRequest{Plan: h.plan(plan), State: stateValue}, &resp)
	return resp
}

func (h *secretSyncTestHarness) gets() int {
	h.backend.mu.Lock()
	defer h.backend.mu.Unlock()
	return h.backend.gets
}

func (h *secretSyncTestHarness) patches() (count int, bodies []map[string]any, statusAtPatch []string) {
	h.backend.mu.Lock()
	defer h.backend.mu.Unlock()
	return h.backend.patches, h.backend.patchBodies, h.backend.statusAtPatch
}

func (h *secretSyncTestHarness) resetCounters() {
	h.backend.mu.Lock()
	defer h.backend.mu.Unlock()
	h.backend.gets, h.backend.patches = 0, 0
}

func expectDiagnosticError(t *testing.T, diags diag.Diagnostics, fragments ...string) {
	t.Helper()
	if !diags.HasError() {
		t.Fatalf("expected an error diagnostic, got: %v", diags)
	}
	text := fmt.Sprint(diags)
	for _, fragment := range fragments {
		if !strings.Contains(text, fragment) {
			t.Errorf("expected the diagnostics to contain %q, got: %s", fragment, text)
		}
	}
}

func TestSecretSyncCreateWithoutWaitDoesNotPoll(t *testing.T) {
	forEachSecretSyncVariant(t, func(t *testing.T, v secretSyncTestVariant) {
		h := newSecretSyncTestHarness(t, v)
		h.backend.nextRun = []fakeSyncStep{{status: infisical.SecretSyncStatusFailed, message: "never read"}}

		resp := h.create(h.model(false, nil))
		if resp.Diagnostics.HasError() {
			t.Fatal(resp.Diagnostics)
		}
		if got := h.gets(); got != 0 {
			t.Errorf("expected no status reads without wait_for_sync_completion, got %d", got)
		}
		if h.stateModel(resp.State).ID.ValueString() == "" {
			t.Error("expected the ID in state")
		}
	})
}

func TestSecretSyncCreateWaitsForFirstSync(t *testing.T) {
	forEachSecretSyncVariant(t, func(t *testing.T, v secretSyncTestVariant) {
		h := newSecretSyncTestHarness(t, v)
		h.backend.nextRun = []fakeSyncStep{
			{status: infisical.SecretSyncStatusPending},
			{status: infisical.SecretSyncStatusRunning},
			{status: infisical.SecretSyncStatusSucceeded},
		}

		resp := h.create(h.model(true, nil))
		if resp.Diagnostics.HasError() {
			t.Fatal(resp.Diagnostics)
		}
		if got := h.gets(); got != 3 {
			t.Errorf("expected Terraform to poll until the sync succeeded (3 reads), got %d", got)
		}
		if !h.stateModel(resp.State).WaitForSyncCompletion.ValueBool() {
			t.Error("expected wait_for_sync_completion to be saved in state")
		}
	})
}

func TestSecretSyncCreateWaitReportsFailedSync(t *testing.T) {
	forEachSecretSyncVariant(t, func(t *testing.T, v secretSyncTestVariant) {
		h := newSecretSyncTestHarness(t, v)
		// A retryable error keeps the status at "running" until the last attempt fails.
		h.backend.nextRun = []fakeSyncStep{
			{status: infisical.SecretSyncStatusRunning},
			{status: infisical.SecretSyncStatusRunning},
			{status: infisical.SecretSyncStatusFailed, message: "Invalid credentials for the destination"},
		}

		resp := h.create(h.model(true, nil))
		expectDiagnosticError(t, resp.Diagnostics, "Invalid credentials for the destination", "tainted", "terraform untaint")

		// Terraform taints the resource only when Create returns state, so the ID must be in it.
		if h.stateModel(resp.State).ID.ValueString() == "" {
			t.Error("expected the created sync to stay in state after a failed wait")
		}
	})
}

func TestSecretSyncCreateWaitTimesOut(t *testing.T) {
	forEachSecretSyncVariant(t, func(t *testing.T, v secretSyncTestVariant) {
		h := newSecretSyncTestHarness(t, v)
		// No steps: the job never leaves the queue, as when no worker picks it up.

		resp := h.create(h.model(true, map[string]string{"create": "50ms"}))
		expectDiagnosticError(t, resp.Diagnostics, "timed out", "last status: pending")
		if h.stateModel(resp.State).ID.ValueString() == "" {
			t.Error("expected the created sync to stay in state after a timeout")
		}
	})
}

func TestSecretSyncCreateWaitSkippedWhenAutoSyncDisabled(t *testing.T) {
	forEachSecretSyncVariant(t, func(t *testing.T, v secretSyncTestVariant) {
		h := newSecretSyncTestHarness(t, v)
		model := h.model(true, nil)
		model.AutoSyncEnabled = types.BoolValue(false)

		resp := h.create(model)
		if resp.Diagnostics.HasError() {
			t.Fatal(resp.Diagnostics)
		}
		if got := h.gets(); got != 0 {
			t.Errorf("expected no wait when auto sync is off, because the API queues no sync; got %d reads", got)
		}
	})
}

func TestSecretSyncWaitReportsDeletedSync(t *testing.T) {
	forEachSecretSyncVariant(t, func(t *testing.T, v secretSyncTestVariant) {
		h := newSecretSyncTestHarness(t, v)
		h.backend.nextRun = []fakeSyncStep{{status: infisical.SecretSyncStatusRunning}, {deleted: true}}

		resp := h.create(h.model(true, nil))
		expectDiagnosticError(t, resp.Diagnostics, "deleted while Terraform was waiting")
	})
}

func TestSecretSyncUpdateWaitsForInFlightSyncBeforePatching(t *testing.T) {
	forEachSecretSyncVariant(t, func(t *testing.T, v secretSyncTestVariant) {
		h := newSecretSyncTestHarness(t, v)
		state := h.applied()
		state.WaitForSyncCompletion = types.BoolValue(true)

		// A job for the old configuration is running. A secret change then sets "pending" while it still runs,
		// so "pending" must not count as idle.
		h.backend.setStatus(state.ID.ValueString(), infisical.SecretSyncStatusRunning,
			fakeSyncStep{status: infisical.SecretSyncStatusPending},
			fakeSyncStep{status: infisical.SecretSyncStatusRunning},
			fakeSyncStep{status: infisical.SecretSyncStatusSucceeded},
		)
		h.backend.nextRun = []fakeSyncStep{
			{status: infisical.SecretSyncStatusRunning},
			{status: infisical.SecretSyncStatusSucceeded},
		}
		h.resetCounters()

		plan := state
		plan.Name = types.StringValue("renamed-sync")
		resp := h.update(state, plan)
		if resp.Diagnostics.HasError() {
			t.Fatal(resp.Diagnostics)
		}

		if _, _, got := h.patches(); len(got) != 1 || got[0] != string(infisical.SecretSyncStatusSucceeded) {
			t.Errorf("expected one PATCH, sent after the in-flight sync settled, got statuses at PATCH time: %v", got)
		}
		if got := h.gets(); got != 5 {
			t.Errorf("expected 3 reads before the PATCH and 2 after it, got %d", got)
		}
		if h.stateModel(resp.State).Name.ValueString() != "renamed-sync" {
			t.Error("expected the updated name in state")
		}
	})
}

func TestSecretSyncUpdateWaitReportsFailedSync(t *testing.T) {
	forEachSecretSyncVariant(t, func(t *testing.T, v secretSyncTestVariant) {
		h := newSecretSyncTestHarness(t, v)
		state := h.applied()
		h.backend.setStatus(state.ID.ValueString(), infisical.SecretSyncStatusSucceeded)
		h.backend.nextRun = []fakeSyncStep{{status: infisical.SecretSyncStatusFailed, message: "Destination not found"}}

		plan := state
		plan.WaitForSyncCompletion = types.BoolValue(true)
		plan.Name = types.StringValue("renamed-sync")
		resp := h.update(state, plan)
		expectDiagnosticError(t, resp.Diagnostics, "was updated, but the sync failed: Destination not found")

		// The update itself was applied, so state must hold the new configuration.
		if h.stateModel(resp.State).Name.ValueString() != "renamed-sync" {
			t.Error("expected the applied update in state after a failed sync")
		}
	})
}

func TestSecretSyncUpdatePreWaitTimeoutDoesNotPatch(t *testing.T) {
	forEachSecretSyncVariant(t, func(t *testing.T, v secretSyncTestVariant) {
		h := newSecretSyncTestHarness(t, v)
		state := h.applied()
		h.backend.setStatus(state.ID.ValueString(), infisical.SecretSyncStatusRunning)
		h.resetCounters()

		plan := state
		plan.WaitForSyncCompletion = types.BoolValue(true)
		plan.Timeouts = secretSyncTimeouts(map[string]string{"update": "50ms"})
		plan.Name = types.StringValue("renamed-sync")
		resp := h.update(state, plan)
		expectDiagnosticError(t, resp.Diagnostics, "already in progress", "did not send the update")

		if count, _, _ := h.patches(); count != 0 {
			t.Errorf("expected no PATCH while the earlier sync was still running, got %d", count)
		}
	})
}

func TestSecretSyncUpdateOnlyProviderSettingsSkipsPatch(t *testing.T) {
	forEachSecretSyncVariant(t, func(t *testing.T, v secretSyncTestVariant) {
		h := newSecretSyncTestHarness(t, v)
		state := h.applied()
		h.resetCounters()

		plan := state
		plan.WaitForSyncCompletion = types.BoolValue(true)
		plan.Timeouts = secretSyncTimeouts(map[string]string{"create": "20m"})
		resp := h.update(state, plan)
		if resp.Diagnostics.HasError() {
			t.Fatal(resp.Diagnostics)
		}

		if count, _, _ := h.patches(); count != 0 || h.gets() != 0 {
			t.Errorf("expected no API calls, because a PATCH would queue a sync; got %d PATCH and %d GET", count, h.gets())
		}
		saved := h.stateModel(resp.State)
		if !saved.WaitForSyncCompletion.ValueBool() || saved.Timeouts.IsNull() {
			t.Errorf("expected the new provider settings in state, got wait=%v timeouts=%v", saved.WaitForSyncCompletion, saved.Timeouts)
		}
	})
}

func TestSecretSyncUpdateSendsAutoSyncDisabled(t *testing.T) {
	forEachSecretSyncVariant(t, func(t *testing.T, v secretSyncTestVariant) {
		h := newSecretSyncTestHarness(t, v)
		state := h.applied()
		state.WaitForSyncCompletion = types.BoolValue(true)
		h.resetCounters()

		plan := state
		plan.AutoSyncEnabled = types.BoolValue(false)
		resp := h.update(state, plan)
		if resp.Diagnostics.HasError() {
			t.Fatal(resp.Diagnostics)
		}

		_, bodies, _ := h.patches()
		if len(bodies) != 1 {
			t.Fatalf("expected one PATCH, got %d", len(bodies))
		}
		value, sent := bodies[0]["isAutoSyncEnabled"]
		if !sent || value != false {
			t.Errorf("expected the PATCH body to send isAutoSyncEnabled=false, got %v (sent: %v)", value, sent)
		}
		if got := h.gets(); got != 0 {
			t.Errorf("expected no wait once auto sync is off, got %d reads", got)
		}
	})
}

func TestSecretSyncReadRemovesMissingOrEmptyID(t *testing.T) {
	forEachSecretSyncVariant(t, func(t *testing.T, v secretSyncTestVariant) {
		h := newSecretSyncTestHarness(t, v)

		for name, id := range map[string]string{"empty ID": "", "deleted outside Terraform": uuid.NewString()} {
			model := h.model(false, nil)
			model.ID = types.StringValue(id)
			state := h.state(model)

			resp := resource.ReadResponse{State: state}
			h.r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("%s: %v", name, resp.Diagnostics)
			}
			if !resp.State.Raw.IsNull() {
				t.Errorf("%s: expected the resource to be removed from state", name)
			}
		}
	})
}

func TestSecretSyncImportSetsWaitDefault(t *testing.T) {
	forEachSecretSyncVariant(t, func(t *testing.T, v secretSyncTestVariant) {
		h := newSecretSyncTestHarness(t, v)
		id := h.applied().ID.ValueString()

		importState := tfsdk.State{Schema: h.schema}
		if diags := importState.Set(context.Background(), &SecretSyncBaseResourceModel{
			SyncOptions:       types.ObjectNull(h.object("sync_options", h.variant.syncOptions).AttributeTypes(context.Background())),
			DestinationConfig: types.ObjectNull(h.object("destination_config", h.variant.destinationConfig).AttributeTypes(context.Background())),
			Timeouts:          secretSyncTimeouts(nil),
		}); diags.HasError() {
			t.Fatal(diags)
		}
		importResp := resource.ImportStateResponse{State: importState}
		h.r.ImportState(context.Background(), resource.ImportStateRequest{ID: id}, &importResp)
		if importResp.Diagnostics.HasError() {
			t.Fatal(importResp.Diagnostics)
		}

		readResp := resource.ReadResponse{State: importResp.State}
		h.r.Read(context.Background(), resource.ReadRequest{State: importResp.State}, &readResp)
		if readResp.Diagnostics.HasError() {
			t.Fatal(readResp.Diagnostics)
		}

		imported := h.stateModel(readResp.State)
		if imported.WaitForSyncCompletion.IsNull() || imported.WaitForSyncCompletion.ValueBool() {
			t.Errorf("expected wait_for_sync_completion to be false after import, so the next plan shows no change; got %v", imported.WaitForSyncCompletion)
		}
		if !imported.Timeouts.IsNull() {
			t.Errorf("expected timeouts to stay null after import, got %v", imported.Timeouts)
		}
		if imported.Name.ValueString() != "my-sync" || imported.Environment.ValueString() != "prod" {
			t.Errorf("expected Read to fill state from the API, got %+v", imported)
		}
	})
}

func TestSecretSyncValidateConfigWarnsWhenAutoSyncDisabled(t *testing.T) {
	forEachSecretSyncVariant(t, func(t *testing.T, v secretSyncTestVariant) {
		h := newSecretSyncTestHarness(t, v)

		for name, tc := range map[string]struct {
			wait     bool
			autoSync types.Bool
			warn     bool
		}{
			"wait with auto sync off":    {wait: true, autoSync: types.BoolValue(false), warn: true},
			"wait with auto sync unset":  {wait: true, autoSync: types.BoolNull()},
			"no wait with auto sync off": {autoSync: types.BoolValue(false)},
		} {
			model := h.model(tc.wait, nil)
			model.AutoSyncEnabled = tc.autoSync
			config := tfsdk.Config{Schema: h.schema, Raw: h.plan(model).Raw}

			var resp resource.ValidateConfigResponse
			h.r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: config}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("%s: %v", name, resp.Diagnostics)
			}
			if got := resp.Diagnostics.WarningsCount() == 1; got != tc.warn {
				t.Errorf("%s: expected warning=%v, got diagnostics: %v", name, tc.warn, resp.Diagnostics)
			}
		}
	})
}

func TestSecretSyncWaitDescriptionMatchesImportSupport(t *testing.T) {
	forEachSecretSyncVariant(t, func(t *testing.T, v secretSyncTestVariant) {
		h := newSecretSyncTestHarness(t, v)
		description := h.schema.Attributes["wait_for_sync_completion"].GetDescription()

		if got := strings.Contains(description, "initial_sync_behavior again"); got != h.r.CanImportSecrets {
			t.Errorf("expected the initial_sync_behavior warning only for destinations that import secrets (CanImportSecrets=%v), got: %s", h.r.CanImportSecrets, description)
		}
		if strings.Contains(description, "—") {
			t.Error("expected no em dash in the description")
		}
	})
}

func TestSecretSyncDefaultTimeoutTextMatchesDuration(t *testing.T) {
	parsed, err := time.ParseDuration(defaultSecretSyncWaitTimeoutText)
	if err != nil || parsed != defaultSecretSyncWaitTimeout {
		t.Errorf("expected the documented default %q to equal %s, got %s (err: %v)", defaultSecretSyncWaitTimeoutText, defaultSecretSyncWaitTimeout, parsed, err)
	}
}

// Status reads share the API's per-IP read rate limit (60/min by default), so the waiter backs off.
func TestSecretSyncPollBackoff(t *testing.T) {
	var waits []time.Duration
	var wait time.Duration
	for range 6 {
		wait = nextSyncPollWait(wait, secretSyncPollInitialInterval, secretSyncPollMaxInterval)
		waits = append(waits, wait)
	}
	want := []time.Duration{5 * time.Second, 10 * time.Second, 20 * time.Second, 30 * time.Second, 30 * time.Second, 30 * time.Second}
	if fmt.Sprint(waits) != fmt.Sprint(want) {
		t.Errorf("expected waits %v, got %v", want, waits)
	}

	var r SecretSyncBaseResource
	if initial, maxInterval := r.syncPollIntervals(); initial != 5*time.Second || maxInterval != 30*time.Second {
		t.Errorf("expected the default backoff to be 5s up to 30s, got %s up to %s", initial, maxInterval)
	}
}

// Right after a create the status is always "pending", so the first read waits one interval.
func TestSecretSyncCreateWaitDoesNotReadImmediately(t *testing.T) {
	forEachSecretSyncVariant(t, func(t *testing.T, v secretSyncTestVariant) {
		h := newSecretSyncTestHarness(t, v)
		h.r.pollInterval = 200 * time.Millisecond
		h.backend.nextRun = []fakeSyncStep{{status: infisical.SecretSyncStatusSucceeded}}

		start := time.Now()
		resp := h.create(h.model(true, nil))
		if resp.Diagnostics.HasError() {
			t.Fatal(resp.Diagnostics)
		}
		if got := h.gets(); got != 1 {
			t.Errorf("expected exactly one status read, got %d", got)
		}
		if elapsed := time.Since(start); elapsed < 200*time.Millisecond {
			t.Errorf("expected the first read to wait one interval, but create returned after %s", elapsed)
		}
	})
}
