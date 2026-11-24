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
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const testApprovalPolicyID = "c1956d8f-d7d4-4ab8-afe3-661a8202bd5c"

// The API returns `bypassers: []` for a policy without bypassers
// (secret-change-policy-fns.ts and access-approval-policy-fns.ts in Infisical core).
func approvalPolicyClient(t *testing.T, wantPath string) *infisical.Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != wantPath {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{"approval": map[string]any{
			"id":               testApprovalPolicyID,
			"projectId":        "project",
			"name":             "build_secrets-changes",
			"secretPath":       "/",
			"approvals":        1,
			"enforcementLevel": "hard",
			"approvers":        []any{},
			"bypassers":        []any{},
		}}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	return &infisical.Client{Config: infisical.Config{
		HttpClient:            resty.New().SetBaseURL(server.URL),
		IsMachineIdentityAuth: true,
	}}
}

func readBypassers(t *testing.T, r resource.Resource, model any) types.Set {
	t.Helper()
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(ctx, model); diags.HasError() {
		t.Fatalf("set state: %v", diags)
	}
	resp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read: %v", resp.Diagnostics)
	}

	var bypassers types.Set
	if diags := resp.State.GetAttribute(ctx, path.Root("bypassers"), &bypassers); diags.HasError() {
		t.Fatalf("get bypassers: %v", diags)
	}
	return bypassers
}

func TestApprovalPolicyReadKeepsEmptyBypassers(t *testing.T) {
	for _, tc := range []struct {
		name      string
		bypassers bool
		wantNull  bool
	}{
		{name: "configured as empty list", bypassers: true, wantNull: false},
		{name: "not configured", bypassers: false, wantNull: true},
	} {
		t.Run("secret/"+tc.name, func(t *testing.T) {
			model := secretApprovalPolicyResourceModel{
				ID:               types.StringValue(testApprovalPolicyID),
				EnvironmentSlugs: types.ListNull(types.StringType),
			}
			if tc.bypassers {
				model.Bypassers = []SecretBypasser{}
			}
			r := &secretApprovalPolicyResource{client: approvalPolicyClient(t, "/api/v1/secret-approvals/"+testApprovalPolicyID)}
			got := readBypassers(t, r, model)
			if got.IsNull() != tc.wantNull || len(got.Elements()) != 0 {
				t.Errorf("bypassers = %v, want null=%v and empty", got, tc.wantNull)
			}
		})
		t.Run("access/"+tc.name, func(t *testing.T) {
			model := accessApprovalPolicyResourceModel{
				ID:               types.StringValue(testApprovalPolicyID),
				EnvironmentSlugs: types.ListNull(types.StringType),
			}
			if tc.bypassers {
				model.Bypassers = []AccessBypasser{}
			}
			r := &accessApprovalPolicyResource{client: approvalPolicyClient(t, "/api/v1/access-approvals/policies/"+testApprovalPolicyID)}
			got := readBypassers(t, r, model)
			if got.IsNull() != tc.wantNull || len(got.Elements()) != 0 {
				t.Errorf("bypassers = %v, want null=%v and empty", got, tc.wantNull)
			}
		})
	}
}
