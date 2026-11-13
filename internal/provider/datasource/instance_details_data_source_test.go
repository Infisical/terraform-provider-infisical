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

// instanceStatusClient returns a client whose /api/status endpoint answers with status and body.
// It uses a non-machine-identity client, because the data source must work with any auth method.
func instanceStatusClient(t *testing.T, status int, body string) *infisical.Client {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return &infisical.Client{Config: infisical.Config{
		HostURL:    srv.URL,
		HttpClient: resty.New().SetBaseURL(srv.URL),
	}}
}

func readInstanceDetails(t *testing.T, client *infisical.Client) datasource.ReadResponse {
	t.Helper()
	ctx := context.Background()

	ds := &InstanceDetailsDataSource{client: client}

	var schemaResp datasource.SchemaResponse
	ds.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("invalid schema: %v", schemaResp.Diagnostics.Errors())
	}
	s := schemaResp.Schema

	objType, ok := s.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatalf("expected the schema to be an object type, got %T", s.Type().TerraformType(ctx))
	}
	config := tfsdk.Config{
		Schema: s,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"aws_account_id": tftypes.NewValue(tftypes.String, nil),
		}),
	}

	resp := datasource.ReadResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(objType, nil)}}
	ds.Read(ctx, datasource.ReadRequest{Config: config}, &resp)

	return resp
}

func TestInstanceDetailsDataSourceReadsAwsAccountId(t *testing.T) {
	client := instanceStatusClient(t, http.StatusOK, `{"date":"2026-11-10T18:34:36.298Z","message":"Ok","awsAccountId":"123456789012"}`)

	resp := readInstanceDetails(t, client)
	if resp.Diagnostics.HasError() {
		t.Fatalf("a successful read must not error, got %v", resp.Diagnostics.Errors())
	}

	var state InstanceDetailsDataSourceModel
	resp.Diagnostics.Append(resp.State.Get(context.Background(), &state)...)
	if resp.Diagnostics.HasError() {
		t.Fatalf("could not read state: %v", resp.Diagnostics.Errors())
	}
	if state.AwsAccountId.ValueString() != "123456789012" {
		t.Errorf("expected aws_account_id 123456789012, got %s", state.AwsAccountId)
	}
}

func TestInstanceDetailsDataSourceLeavesAwsAccountIdNullWhenAbsent(t *testing.T) {
	client := instanceStatusClient(t, http.StatusOK, `{"date":"2026-11-10T18:34:36.298Z","message":"Ok"}`)

	resp := readInstanceDetails(t, client)
	if resp.Diagnostics.HasError() {
		t.Fatalf("a successful read must not error, got %v", resp.Diagnostics.Errors())
	}

	var state InstanceDetailsDataSourceModel
	resp.Diagnostics.Append(resp.State.Get(context.Background(), &state)...)
	if resp.Diagnostics.HasError() {
		t.Fatalf("could not read state: %v", resp.Diagnostics.Errors())
	}
	if !state.AwsAccountId.IsNull() {
		t.Errorf("expected aws_account_id to be null, got %s", state.AwsAccountId)
	}
}

func TestInstanceDetailsDataSourceReadLeavesStateUnsetOnAPIError(t *testing.T) {
	client := instanceStatusClient(t, http.StatusServiceUnavailable, `{"message":"Service unavailable"}`)

	resp := readInstanceDetails(t, client)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error diagnostic for a 503 response")
	}
	if !resp.State.Raw.IsNull() {
		t.Error("expected state to stay unset after a failed read")
	}
}
