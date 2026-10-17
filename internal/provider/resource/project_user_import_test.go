package resource

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Import must work against the older API without inviting a user, changing roles,
// or requiring a different import ID format.
func TestProjectUserImportLegacyAPI(t *testing.T) {
	const projectID = "11111111-1111-4111-8111-111111111111"
	const userID = "22222222-2222-4222-8222-222222222222"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.Method + " " + req.URL.Path {
		case "GET /api/v1/projects/" + projectID + "/user/" + userID + "/membership":
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"statusCode":404,"message":"Route GET:%s not found"}`, req.URL.Path)
		case "GET /api/v1/projects/" + projectID + "/memberships":
			fmt.Fprintf(w, `{"memberships":[{"id":"membership-1","userId":%q,"user":{"id":%q,"username":"reader"}}]}`, userID, userID)
		case "POST /api/v1/workspace/" + projectID + "/memberships/details":
			fmt.Fprintf(w, `{"membership":{"id":"membership-1","userId":%q,"user":{"id":%q,"username":"reader","email":"reader@example.com"},"roles":[{"id":"role-1","role":"custom","customRoleSlug":"accessor","customRoleId":"custom-1","isTemporary":false}]}}`, userID, userID)
		default:
			t.Errorf("unexpected or mutating request: %s %s", req.Method, req.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	r := &ProjectUserResource{client: &infisical.Client{Config: infisical.Config{
		HttpClient: resty.New().SetBaseURL(server.URL), IsMachineIdentityAuth: true,
	}}}
	ctx := context.Background()
	var schema resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schema)
	resp := resource.ImportStateResponse{State: tfsdk.State{Schema: schema.Schema}}
	if diags := resp.State.Set(ctx, &ProjectUserResourceModel{User: types.ObjectNull(map[string]attr.Type{
		"id": types.StringType, "email": types.StringType, "first_name": types.StringType, "last_name": types.StringType,
	})}); diags.HasError() {
		t.Fatal(diags)
	}
	r.ImportState(ctx, resource.ImportStateRequest{ID: projectID + "," + userID}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("import failed: %v", resp.Diagnostics)
	}
	var state ProjectUserResourceModel
	if diags := resp.State.Get(ctx, &state); diags.HasError() {
		t.Fatal(diags)
	}
	if state.ProjectID.ValueString() != projectID || state.Username.ValueString() != "reader" || state.MembershipId.ValueString() != "membership-1" {
		t.Fatalf("incorrect imported identity: %+v", state)
	}
	if len(state.Roles) != 1 || state.Roles[0].RoleSlug.ValueString() != "accessor" || state.Roles[0].CustomRoleID.ValueString() != "custom-1" || state.Roles[0].IsTemporary.ValueBool() {
		t.Fatalf("import changed existing roles: %+v", state.Roles)
	}
}
