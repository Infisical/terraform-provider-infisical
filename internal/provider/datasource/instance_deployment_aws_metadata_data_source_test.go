package datasource

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// awsMetadataClient returns a client whose AWS deployment metadata endpoint answers with status and body.
func awsMetadataClient(t *testing.T, status int, body string) *infisical.Client {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/instance/deployment/aws/metadata", func(w http.ResponseWriter, _ *http.Request) {
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

func readAwsMetadata(t *testing.T, client *infisical.Client) datasource.ReadResponse {
	t.Helper()
	ctx := context.Background()

	ds := &InstanceDeploymentAwsMetadataDataSource{client: client}

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
			"account_id": tftypes.NewValue(tftypes.String, nil),
		}),
	}

	resp := datasource.ReadResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(objType, nil)}}
	ds.Read(ctx, datasource.ReadRequest{Config: config}, &resp)

	return resp
}

func readAwsMetadataState(t *testing.T, resp datasource.ReadResponse) InstanceDeploymentAwsMetadataDataSourceModel {
	t.Helper()

	if resp.Diagnostics.HasError() {
		t.Fatalf("a successful read must not error, got %v", resp.Diagnostics.Errors())
	}
	var state InstanceDeploymentAwsMetadataDataSourceModel
	resp.Diagnostics.Append(resp.State.Get(context.Background(), &state)...)
	if resp.Diagnostics.HasError() {
		t.Fatalf("could not read state: %v", resp.Diagnostics.Errors())
	}
	return state
}

func TestInstanceDeploymentAwsMetadataDataSourceReadsAccountId(t *testing.T) {
	client := awsMetadataClient(t, http.StatusOK, `{"metadata":{"accountId":"123456789012"}}`)

	state := readAwsMetadataState(t, readAwsMetadata(t, client))
	if state.AccountId.ValueString() != "123456789012" {
		t.Errorf("expected account_id 123456789012, got %s", state.AccountId)
	}
}

func TestInstanceDeploymentAwsMetadataDataSourceLeavesAccountIdNullWithoutAwsIdentity(t *testing.T) {
	client := awsMetadataClient(t, http.StatusOK, `{"metadata":{"accountId":null}}`)

	state := readAwsMetadataState(t, readAwsMetadata(t, client))
	if !state.AccountId.IsNull() {
		t.Errorf("expected account_id to be null, got %s", state.AccountId)
	}
}

func TestInstanceDeploymentAwsMetadataDataSourceExplainsOlderServers(t *testing.T) {
	client := awsMetadataClient(t, http.StatusOK, `{}`)
	client.Config.HttpClient = resty.New().SetBaseURL(client.Config.HostURL + "/missing")

	resp := readAwsMetadata(t, client)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error diagnostic when the server has no such endpoint")
	}
	if summary := resp.Diagnostics.Errors()[0].Summary(); !strings.Contains(summary, "not available on this Infisical server") {
		t.Errorf("expected the upgrade message, got %q", summary)
	}
	if !resp.State.Raw.IsNull() {
		t.Error("expected state to stay unset after a failed read")
	}
}

func TestInstanceDeploymentAwsMetadataDataSourceLeavesStateUnsetOnAPIError(t *testing.T) {
	client := awsMetadataClient(t, http.StatusUnauthorized, `{"message":"Unauthorized"}`)

	resp := readAwsMetadata(t, client)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error diagnostic for a 401 response")
	}
	if !resp.State.Raw.IsNull() {
		t.Error("expected state to stay unset after a failed read")
	}
}

func TestInstanceDeploymentAwsMetadataDataSourceRequiresMachineIdentity(t *testing.T) {
	client := awsMetadataClient(t, http.StatusOK, `{"metadata":{"accountId":"123456789012"}}`)
	client.Config.IsMachineIdentityAuth = false

	resp := readAwsMetadata(t, client)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error diagnostic without machine identity auth")
	}
}
