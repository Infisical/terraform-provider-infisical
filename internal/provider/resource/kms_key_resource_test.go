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
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func kmsKeyTestSchema(t *testing.T) schema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	NewKMSKeyResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func kmsKeyExportabilityAttribute(t *testing.T, resourceSchema schema.Schema) schema.BoolAttribute {
	t.Helper()
	attribute, ok := resourceSchema.Attributes["is_exportable"].(schema.BoolAttribute)
	if !ok || !attribute.Optional || !attribute.Computed {
		t.Fatal("is_exportable must be an optional, computed boolean")
	}

	if attribute.Default != nil {
		t.Fatal("is_exportable must not declare a static default")
	}
	return attribute
}

func newKMSKeyTestResource(t *testing.T, handler http.HandlerFunc) *kmsKeyResource {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &kmsKeyResource{client: &infisical.Client{Config: infisical.Config{
		HttpClient: resty.New().SetBaseURL(server.URL),
	}}}
}

func kmsKeyTestPlan(t *testing.T, ctx context.Context, resourceSchema schema.Schema, model kmsKeyResourceModel) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: resourceSchema}
	if diags := plan.Set(ctx, &model); diags.HasError() {
		t.Fatal(diags)
	}
	return plan
}

func TestKMSKeyExportabilityPlanModifiers(t *testing.T) {
	ctx := context.Background()
	resourceSchema := kmsKeyTestSchema(t)
	attribute := kmsKeyExportabilityAttribute(t, resourceSchema)

	tests := []struct {
		name            string
		config          types.Bool
		state           types.Bool
		plan            types.Bool
		wantPlan        types.Bool
		wantReplacement bool
	}{
		{
			// A key created outside Terraform with isExportable=false and then imported.
			name:     "unconfigured_keeps_non_exportable_key",
			config:   types.BoolNull(),
			state:    types.BoolValue(false),
			plan:     types.BoolValue(false),
			wantPlan: types.BoolValue(false),
		},
		{
			// State written before this attribute existed, planned with -refresh=false. The value is resolved by
			// the apply, which must not be a replacement.
			name:     "unconfigured_tolerates_missing_state",
			config:   types.BoolNull(),
			state:    types.BoolNull(),
			plan:     types.BoolUnknown(),
			wantPlan: types.BoolUnknown(),
		},
		{
			name:            "configured_change_replaces",
			config:          types.BoolValue(false),
			state:           types.BoolValue(true),
			plan:            types.BoolValue(false),
			wantPlan:        types.BoolValue(false),
			wantReplacement: true,
		},
		{
			name:     "configured_match_is_kept",
			config:   types.BoolValue(false),
			state:    types.BoolValue(false),
			plan:     types.BoolValue(false),
			wantPlan: types.BoolValue(false),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configState := tfsdk.State{Schema: resourceSchema}
			if diags := configState.Set(ctx, &kmsKeyResourceModel{IsExportable: test.config}); diags.HasError() {
				t.Fatal(diags)
			}
			config := tfsdk.Config{Schema: resourceSchema, Raw: configState.Raw}
			state := tfsdk.State{Schema: resourceSchema}
			if diags := state.Set(ctx, &kmsKeyResourceModel{IsExportable: test.state}); diags.HasError() {
				t.Fatal(diags)
			}
			plan := kmsKeyTestPlan(t, ctx, resourceSchema, kmsKeyResourceModel{IsExportable: test.plan})

			planValue := test.plan
			requiresReplace := false
			for _, modifier := range attribute.PlanModifiers {
				modifierResp := planmodifier.BoolResponse{PlanValue: planValue}
				modifier.PlanModifyBool(ctx, planmodifier.BoolRequest{
					Path:        path.Root("is_exportable"),
					Config:      config,
					ConfigValue: test.config,
					State:       state,
					StateValue:  test.state,
					Plan:        plan,
					PlanValue:   planValue,
				}, &modifierResp)
				if modifierResp.Diagnostics.HasError() {
					t.Fatal(modifierResp.Diagnostics)
				}
				planValue = modifierResp.PlanValue
				requiresReplace = requiresReplace || modifierResp.RequiresReplace
			}

			if !planValue.Equal(test.wantPlan) {
				t.Errorf("planned is_exportable = %v, want %v", planValue, test.wantPlan)
			}
			if requiresReplace != test.wantReplacement {
				t.Errorf("RequiresReplace = %t, want %t", requiresReplace, test.wantReplacement)
			}
		})
	}
}

func TestKMSKeyExportabilityLifecycle(t *testing.T) {
	for _, exportable := range []bool{false, true} {
		t.Run(fmt.Sprintf("exportable_%t", exportable), func(t *testing.T) {
			ctx := context.Background()
			resourceSchema := kmsKeyTestSchema(t)
			name := "test-key"
			r := newKMSKeyTestResource(t, func(w http.ResponseWriter, req *http.Request) {
				switch req.Method + " " + req.URL.Path {
				case "POST /api/v1/kms/keys":
					var body map[string]json.RawMessage
					if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
						t.Error(err)
						http.Error(w, "invalid JSON", http.StatusBadRequest)
						return
					}
					// Omitting false would silently enable exports via the API default.
					if got := string(body["isExportable"]); got != fmt.Sprint(exportable) {
						t.Errorf("POST isExportable = %q, want %t", got, exportable)
					}
				case "GET /api/v1/kms/keys/key-1":
				case "PATCH /api/v1/kms/keys/key-1":
					var body map[string]json.RawMessage
					if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
						t.Error(err)
						http.Error(w, "invalid JSON", http.StatusBadRequest)
						return
					}
					if _, exists := body["isExportable"]; exists {
						t.Error("PATCH must not try to change immutable exportability")
					}
					if got := string(body["name"]); got != `"renamed-key"` {
						t.Errorf("PATCH name = %q, want renamed-key", got)
					}
					name = "renamed-key"
				default:
					t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
					http.NotFound(w, req)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprintf(w, `{"key":{"id":"key-1","projectId":"project-1","name":%q,"isExportable":%t,"isDisabled":false,"keyUsage":"encrypt-decrypt","encryptionAlgorithm":"aes-256-gcm"}}`, name, exportable)
			})

			plan := kmsKeyTestPlan(t, ctx, resourceSchema, kmsKeyResourceModel{
				ProjectId:           types.StringValue("project-1"),
				Name:                types.StringValue("test-key"),
				Description:         types.StringValue(""),
				KeyUsage:            types.StringValue(ENCRYPTION_KEY_USAGE),
				EncryptionAlgorithm: types.StringValue(ENCRYPTION_ALGORITHM_AES_256_GCM),
				IsDisabled:          types.BoolValue(false),
				IsExportable:        types.BoolValue(exportable),
			})
			createResp := resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema}}
			r.Create(ctx, resource.CreateRequest{Plan: plan}, &createResp)
			if createResp.Diagnostics.HasError() {
				t.Fatal(createResp.Diagnostics)
			}
			assertKMSKeyExportability(t, createResp.State, exportable)

			// Refresh must replace stale state with the value returned by the API.
			staleState := createResp.State
			if diags := staleState.SetAttribute(ctx, path.Root("is_exportable"), !exportable); diags.HasError() {
				t.Fatal(diags)
			}
			readResp := resource.ReadResponse{State: staleState}
			r.Read(ctx, resource.ReadRequest{State: staleState}, &readResp)
			if readResp.Diagnostics.HasError() {
				t.Fatal(readResp.Diagnostics)
			}
			assertKMSKeyExportability(t, readResp.State, exportable)

			// Imports start with only an ID, so Read must populate every attribute the configuration can set.
			// A null project_id would force a replacement on the first apply after the import.
			importState := tfsdk.State{Schema: resourceSchema}
			if diags := importState.Set(ctx, &kmsKeyResourceModel{}); diags.HasError() {
				t.Fatal(diags)
			}
			importResp := resource.ImportStateResponse{State: importState}
			r.ImportState(ctx, resource.ImportStateRequest{ID: "key-1"}, &importResp)
			if importResp.Diagnostics.HasError() {
				t.Fatal(importResp.Diagnostics)
			}
			readResp = resource.ReadResponse{State: importResp.State}
			r.Read(ctx, resource.ReadRequest{State: importResp.State}, &readResp)
			if readResp.Diagnostics.HasError() {
				t.Fatal(readResp.Diagnostics)
			}
			assertKMSKeyExportability(t, readResp.State, exportable)
			assertKMSKeyAttribute(t, readResp.State, "project_id", types.StringValue("project-1"))

			updatePlan := tfsdk.Plan{Schema: resourceSchema, Raw: createResp.State.Raw}
			if diags := updatePlan.SetAttribute(ctx, path.Root("name"), "renamed-key"); diags.HasError() {
				t.Fatal(diags)
			}
			updateResp := resource.UpdateResponse{State: createResp.State}
			r.Update(ctx, resource.UpdateRequest{Plan: updatePlan, State: createResp.State}, &updateResp)
			if updateResp.Diagnostics.HasError() {
				t.Fatal(updateResp.Diagnostics)
			}
			assertKMSKeyExportability(t, updateResp.State, exportable)
		})
	}
}

// Instances older than Infisical v0.161.1 do not know about isExportable: they omit it from every response, and
// the provider must read that as the exportable keys those instances create, not as false.
func TestKMSKeyExportabilityOnLegacyInstance(t *testing.T) {
	ctx := context.Background()
	resourceSchema := kmsKeyTestSchema(t)

	legacyHandler := func(t *testing.T, assertCreateBody func(map[string]json.RawMessage)) http.HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodPost {
				var body map[string]json.RawMessage
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					t.Error(err)
					http.Error(w, "invalid JSON", http.StatusBadRequest)
					return
				}
				assertCreateBody(body)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"key":{"id":"key-1","projectId":"project-1","name":"test-key","isDisabled":false,"keyUsage":"encrypt-decrypt","encryptionAlgorithm":"aes-256-gcm"}}`)
		}
	}

	basePlan := kmsKeyResourceModel{
		ProjectId:           types.StringValue("project-1"),
		Name:                types.StringValue("test-key"),
		Description:         types.StringValue(""),
		KeyUsage:            types.StringValue(ENCRYPTION_KEY_USAGE),
		EncryptionAlgorithm: types.StringValue(ENCRYPTION_ALGORITHM_AES_256_GCM),
		IsDisabled:          types.BoolValue(false),
	}

	t.Run("unconfigured_create_reports_exportable", func(t *testing.T) {
		r := newKMSKeyTestResource(t, legacyHandler(t, func(body map[string]json.RawMessage) {
			if _, exists := body["isExportable"]; exists {
				t.Error("POST must omit isExportable when it is not configured")
			}
		}))

		model := basePlan
		model.IsExportable = types.BoolUnknown()
		createResp := resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema}}
		r.Create(ctx, resource.CreateRequest{Plan: kmsKeyTestPlan(t, ctx, resourceSchema, model)}, &createResp)
		if createResp.Diagnostics.HasError() {
			t.Fatal(createResp.Diagnostics)
		}
		assertKMSKeyExportability(t, createResp.State, true)
	})

	t.Run("read_reports_exportable", func(t *testing.T) {
		r := newKMSKeyTestResource(t, legacyHandler(t, func(map[string]json.RawMessage) {}))

		state := tfsdk.State{Schema: resourceSchema}
		if diags := state.Set(ctx, &kmsKeyResourceModel{ID: types.StringValue("key-1")}); diags.HasError() {
			t.Fatal(diags)
		}
		readResp := resource.ReadResponse{State: state}
		r.Read(ctx, resource.ReadRequest{State: state}, &readResp)
		if readResp.Diagnostics.HasError() {
			t.Fatal(readResp.Diagnostics)
		}
		assertKMSKeyExportability(t, readResp.State, true)
	})

	// Every key on such an instance is exportable, so an explicit is_exportable = true is already satisfied.
	t.Run("configured_exportable_create_succeeds", func(t *testing.T) {
		r := newKMSKeyTestResource(t, func(w http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodDelete {
				t.Error("a key that matches the configuration must not be deleted")
			}
			legacyHandler(t, func(body map[string]json.RawMessage) {
				if got := string(body["isExportable"]); got != "true" {
					t.Errorf("POST isExportable = %q, want true", got)
				}
			})(w, req)
		})

		model := basePlan
		model.IsExportable = types.BoolValue(true)
		createResp := resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema}}
		r.Create(ctx, resource.CreateRequest{Plan: kmsKeyTestPlan(t, ctx, resourceSchema, model)}, &createResp)
		if createResp.Diagnostics.HasError() {
			t.Fatal(createResp.Diagnostics)
		}
		assertKMSKeyExportability(t, createResp.State, true)
	})

	t.Run("configured_create_deletes_the_exportable_key", func(t *testing.T) {
		deleted := false
		r := newKMSKeyTestResource(t, func(w http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodDelete {
				if req.URL.Path != "/api/v1/kms/keys/key-1" {
					t.Errorf("DELETE %s, want /api/v1/kms/keys/key-1", req.URL.Path)
				}
				deleted = true
			}
			legacyHandler(t, func(map[string]json.RawMessage) {})(w, req)
		})

		model := basePlan
		model.IsExportable = types.BoolValue(false)
		createResp := resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema}}
		r.Create(ctx, resource.CreateRequest{Plan: kmsKeyTestPlan(t, ctx, resourceSchema, model)}, &createResp)
		if !createResp.Diagnostics.HasError() {
			t.Fatal("creating a non-exportable key against an instance that ignores isExportable must fail")
		}
		if detail := createResp.Diagnostics.Errors()[0].Detail(); !strings.Contains(detail, "does not support is_exportable") {
			t.Errorf("unexpected diagnostic: %s", detail)
		}
		if !deleted {
			t.Error("the exportable key must be deleted again")
		}
		// The key is gone, so Terraform must not track it.
		if !createResp.State.Raw.IsNull() {
			t.Errorf("state = %v, want no state for a key that was deleted", createResp.State.Raw)
		}
	})

	t.Run("configured_create_keeps_state_when_cleanup_fails", func(t *testing.T) {
		r := newKMSKeyTestResource(t, func(w http.ResponseWriter, req *http.Request) {
			if req.Method == http.MethodDelete {
				http.Error(w, `{"message":"delete protection enabled"}`, http.StatusBadRequest)
				return
			}
			legacyHandler(t, func(map[string]json.RawMessage) {})(w, req)
		})

		model := basePlan
		model.IsExportable = types.BoolValue(false)
		createResp := resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema}}
		r.Create(ctx, resource.CreateRequest{Plan: kmsKeyTestPlan(t, ctx, resourceSchema, model)}, &createResp)
		if !createResp.Diagnostics.HasError() {
			t.Fatal("creating a non-exportable key against an instance that ignores isExportable must fail")
		}
		// The key outlived the failed delete, so it has to land in state instead of leaking.
		assertKMSKeyAttribute(t, createResp.State, "id", types.StringValue("key-1"))
		assertKMSKeyExportability(t, createResp.State, true)
	})
}

func assertKMSKeyExportability(t *testing.T, state tfsdk.State, want bool) {
	t.Helper()
	var got types.Bool
	if diags := state.GetAttribute(context.Background(), path.Root("is_exportable"), &got); diags.HasError() {
		t.Fatal(diags)
	}
	if !got.Equal(types.BoolValue(want)) {
		t.Errorf("is_exportable = %v, want %t", got, want)
	}
}

func assertKMSKeyAttribute(t *testing.T, state tfsdk.State, name string, want types.String) {
	t.Helper()
	var got types.String
	if diags := state.GetAttribute(context.Background(), path.Root(name), &got); diags.HasError() {
		t.Fatal(diags)
	}
	if !got.Equal(want) {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}
