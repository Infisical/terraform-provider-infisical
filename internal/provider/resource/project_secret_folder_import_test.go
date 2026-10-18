package resource

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestSecretFolderImportDefaults prevents an API update caused only by a missing local default.
func TestSecretFolderImportDefaults(t *testing.T) {
	for _, tc := range []struct {
		name, fullPath, parent string
		description            any
		wantDescription        string
	}{
		{"root-child", "/example", "/", nil, ""},
		{"nested-child", "/parent/example", "/parent", "Existing description", "Existing description"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				calls++
				if req.Method != http.MethodGet || req.URL.Path != "/api/v1/folders/folder-id" {
					t.Errorf("import and refresh must only GET the folder: %s %s", req.Method, req.URL.Path)
					http.Error(w, "unexpected request", http.StatusMethodNotAllowed)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(map[string]any{"folder": map[string]any{
					"id": "folder-id", "name": "example", "projectId": "project-id",
					"envId": "environment-id", "path": tc.fullPath, "description": tc.description,
					"environment": map[string]any{"envSlug": "dev"},
				}}); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			r := &projectSecretFolderResource{client: &infisical.Client{Config: infisical.Config{
				IsMachineIdentityAuth: true, HttpClient: resty.New().SetBaseURL(server.URL),
			}}}
			schemaResp := resource.SchemaResponse{}
			r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
			resp := resource.ImportStateResponse{State: tfsdk.State{
				Schema: schemaResp.Schema,
				Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
			}}
			r.ImportState(ctx, resource.ImportStateRequest{ID: "folder-id"}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			expected := map[string]any{
				"id": "folder-id", "name": "example", "project_id": "project-id",
				"environment_slug": "dev", "folder_path": tc.parent, "path": tc.fullPath,
				"description": tc.wantDescription, "force_delete": types.BoolValue(false),
			}
			assertSecretFolderImportAttributes(t, resp.State, expected)
			readResp := resource.ReadResponse{State: resp.State}
			r.Read(ctx, resource.ReadRequest{State: resp.State}, &readResp)
			if readResp.Diagnostics.HasError() {
				t.Fatal(readResp.Diagnostics)
			}
			expected["environment_id"] = "environment-id"
			assertSecretFolderImportAttributes(t, readResp.State, expected)
			// Read must not reset an explicitly enabled deletion policy on existing resources.
			if diags := readResp.State.SetAttribute(ctx, path.Root("force_delete"), true); diags.HasError() {
				t.Fatal(diags)
			}
			secondRead := resource.ReadResponse{State: readResp.State}
			r.Read(ctx, resource.ReadRequest{State: readResp.State}, &secondRead)
			if secondRead.Diagnostics.HasError() {
				t.Fatal(secondRead.Diagnostics)
			}
			expected["force_delete"] = types.BoolValue(true)
			assertSecretFolderImportAttributes(t, secondRead.State, expected)
			if calls != 3 {
				t.Errorf("expected three GET requests, got %d", calls)
			}
		})
	}
}

// assertSecretFolderImportAttributes checks import and refresh without depending on other resource tests.
func assertSecretFolderImportAttributes(t *testing.T, state tfsdk.State, expected map[string]any) {
	t.Helper()
	for name, want := range expected {
		got := reflect.New(reflect.TypeOf(want))
		if diags := state.GetAttribute(context.Background(), path.Root(name), got.Interface()); diags.HasError() {
			t.Fatalf("%s: %v", name, diags)
		}
		if !reflect.DeepEqual(got.Elem().Interface(), want) {
			t.Errorf("%s: got %#v, want %#v", name, got.Elem().Interface(), want)
		}
	}
}
