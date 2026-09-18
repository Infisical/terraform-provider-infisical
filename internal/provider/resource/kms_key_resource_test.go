package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
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

func TestKMSKeyExportabilityPlan(t *testing.T) {
	ctx := context.Background()
	resourceSchema := kmsKeyTestSchema(t)
	attribute, ok := resourceSchema.Attributes["is_exportable"].(schema.BoolAttribute)
	if !ok || !attribute.Optional || !attribute.Computed || attribute.Default == nil {
		t.Fatal("is_exportable must be an optional boolean with a default")
	}

	var defaultResp defaults.BoolResponse
	attribute.Default.DefaultBool(ctx, defaults.BoolRequest{Path: path.Root("is_exportable")}, &defaultResp)
	if defaultResp.Diagnostics.HasError() || !defaultResp.PlanValue.Equal(types.BoolValue(true)) {
		t.Fatalf("omitting is_exportable must preserve the API default of true: %+v", defaultResp)
	}

	for _, before := range []bool{false, true} {
		for _, after := range []bool{false, true} {
			t.Run(fmt.Sprintf("%t_to_%t", before, after), func(t *testing.T) {
				state := tfsdk.State{Schema: resourceSchema}
				if diags := state.Set(ctx, &kmsKeyResourceModel{IsExportable: types.BoolValue(before)}); diags.HasError() {
					t.Fatal(diags)
				}
				plan := tfsdk.Plan{Schema: resourceSchema}
				if diags := plan.Set(ctx, &kmsKeyResourceModel{IsExportable: types.BoolValue(after)}); diags.HasError() {
					t.Fatal(diags)
				}

				requiresReplace := false
				for _, modifier := range attribute.PlanModifiers {
					var resp planmodifier.BoolResponse
					modifier.PlanModifyBool(ctx, planmodifier.BoolRequest{
						Path:       path.Root("is_exportable"),
						State:      state,
						StateValue: types.BoolValue(before),
						Plan:       plan,
						PlanValue:  types.BoolValue(after),
					}, &resp)
					if resp.Diagnostics.HasError() {
						t.Fatal(resp.Diagnostics)
					}
					requiresReplace = requiresReplace || resp.RequiresReplace
				}
				if requiresReplace != (before != after) {
					t.Errorf("RequiresReplace = %t for is_exportable changing from %t to %t", requiresReplace, before, after)
				}
			})
		}
	}
}

func TestKMSKeyExportabilityLifecycle(t *testing.T) {
	for _, exportable := range []bool{false, true} {
		t.Run(fmt.Sprintf("exportable_%t", exportable), func(t *testing.T) {
			ctx := context.Background()
			resourceSchema := kmsKeyTestSchema(t)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				name := "test-key"
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
			}))
			t.Cleanup(server.Close)
			r := &kmsKeyResource{client: &infisical.Client{Config: infisical.Config{
				HttpClient: resty.New().SetBaseURL(server.URL),
			}}}

			plan := tfsdk.Plan{Schema: resourceSchema}
			if diags := plan.Set(ctx, &kmsKeyResourceModel{
				ProjectId:           types.StringValue("project-1"),
				Name:                types.StringValue("test-key"),
				Description:         types.StringValue(""),
				KeyUsage:            types.StringValue(ENCRYPTION_KEY_USAGE),
				EncryptionAlgorithm: types.StringValue(ENCRYPTION_ALGORITHM_AES_256_GCM),
				IsDisabled:          types.BoolValue(false),
				IsExportable:        types.BoolValue(exportable),
			}); diags.HasError() {
				t.Fatal(diags)
			}
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

			// Imports start with only an ID, so Read must populate exportability.
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
