package datasource

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// groupsClient returns a client whose groups endpoint answers with status and body.
func groupsClient(t *testing.T, status int, body string) *infisical.Client {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/groups", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return &infisical.Client{Config: infisical.Config{
		HostURL:               srv.URL,
		HttpClient:            resty.New().SetBaseURL(srv.URL),
		IsMachineIdentityAuth: true,
	}}
}

// readGroups runs the data source's Read the way Terraform would: an empty config in,
// a null state that Read is expected to fill.
func readGroups(t *testing.T, client *infisical.Client) datasource.ReadResponse {
	t.Helper()
	ctx := context.Background()

	ds := &GroupsDataSource{client: client}

	var schemaResp datasource.SchemaResponse
	ds.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("invalid schema: %v", schemaResp.Diagnostics.Errors())
	}
	s := schemaResp.Schema

	objType := s.Type().TerraformType(ctx).(tftypes.Object)
	config := tfsdk.Config{
		Schema: s,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"groups": tftypes.NewValue(objType.AttributeTypes["groups"], nil),
		}),
	}

	resp := datasource.ReadResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(objType, nil)}}
	ds.Read(ctx, datasource.ReadRequest{Config: config}, &resp)

	return resp
}

// Going through the schema and state, rather than inspecting the mapping alone, is what
// catches the model, the schema and the list's attribute types drifting apart.
func TestGroupsDataSourceReadExposesSlug(t *testing.T) {
	client := groupsClient(t, http.StatusOK, `[
		{"id":"11111111-1111-1111-1111-111111111111","orgId":"22222222-2222-2222-2222-222222222222","name":"Platform Engineers","slug":"platform-engineers","role":"member","roleId":"33333333-3333-3333-3333-333333333333"}
	]`)

	resp := readGroups(t, client)

	if resp.Diagnostics.HasError() {
		t.Fatalf("a successful read must not error, got %v", resp.Diagnostics.Errors())
	}

	ctx := context.Background()
	var data GroupsDataSourceModel
	if diags := resp.State.Get(ctx, &data); diags.HasError() {
		t.Fatalf("state does not match the model: %v", diags.Errors())
	}

	var groups []InfisicalGroupDetails
	if diags := data.Groups.ElementsAs(ctx, &groups, false); diags.HasError() {
		t.Fatalf("groups do not match the model: %v", diags.Errors())
	}

	if len(groups) != 1 {
		t.Fatalf("expected one group, got %d", len(groups))
	}
	g := groups[0]
	if g.Slug.ValueString() != "platform-engineers" {
		t.Errorf("expected slug %q in state, got %q", "platform-engineers", g.Slug.ValueString())
	}
	if g.ID.ValueString() != "11111111-1111-1111-1111-111111111111" || g.Name.ValueString() != "Platform Engineers" {
		t.Errorf("unexpected group: %+v", g)
	}
	if g.OrgID.ValueString() != "22222222-2222-2222-2222-222222222222" || g.Role.ValueString() != "member" || g.RoleId.ValueString() != "33333333-3333-3333-3333-333333333333" {
		t.Errorf("unexpected group: %+v", g)
	}
}

// A failed fetch says nothing about which groups exist. Writing an empty list to state
// alongside the error would present the failure as an organization with no groups.
// Read guards this twice (the early return after the fetch and the HasError check
// before State.Set), so this fails only if both are lost.
func TestGroupsDataSourceReadLeavesStateUnsetOnAPIError(t *testing.T) {
	client := groupsClient(t, http.StatusForbidden,
		`{"statusCode":403,"message":"You do not have permission to read groups"}`)

	resp := readGroups(t, client)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error when the API rejects the request")
	}
	if !resp.State.Raw.IsNull() {
		t.Errorf("state must be left unset when the fetch fails, got %v", resp.State.Raw)
	}
}
