package datasource

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// secretsServer answers the raw secrets endpoint and records the include_imports query value
// of the last request together with the number of requests received.
func secretsServer(t *testing.T, authStrategy infisical.AuthStrategyType, machineIdentity bool) (*infisical.Client, *string, *atomic.Int32) {
	t.Helper()

	var sent string
	var calls atomic.Int32

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		sent = r.URL.Query().Get("include_imports")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"secrets":[{"secretKey":"KEY","secretValue":"value"}]}`))
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return &infisical.Client{Config: infisical.Config{
		HostURL:               srv.URL,
		HttpClient:            resty.New().SetBaseURL(srv.URL),
		AuthStrategy:          authStrategy,
		IsMachineIdentityAuth: machineIdentity,
	}}, &sent, &calls
}

// readSecrets runs the data source's Read the way Terraform would. includeImports nil means the
// attribute is omitted from the configuration.
func readSecrets(t *testing.T, client *infisical.Client, includeImports *bool) datasource.ReadResponse {
	t.Helper()
	ctx := context.Background()

	ds := &SecretsDataSource{client: client}

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

	include := tftypes.NewValue(tftypes.Bool, nil)
	if includeImports != nil {
		include = tftypes.NewValue(tftypes.Bool, *includeImports)
	}

	config := tfsdk.Config{
		Schema: s,
		Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
			"folder_path":     tftypes.NewValue(tftypes.String, "/"),
			"env_slug":        tftypes.NewValue(tftypes.String, "dev"),
			"workspace_id":    tftypes.NewValue(tftypes.String, "project-id"),
			"include_imports": include,
			"secrets":         tftypes.NewValue(objType.AttributeTypes["secrets"], nil),
		}),
	}

	resp := datasource.ReadResponse{State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(objType, nil)}}
	ds.Read(ctx, datasource.ReadRequest{Config: config}, &resp)

	return resp
}

func boolPtr(b bool) *bool { return &b }

func TestSecretsDataSourceReadForwardsIncludeImports(t *testing.T) {
	cases := []struct {
		name   string
		config *bool
		want   string
	}{
		{"omitted defaults to false", nil, "false"},
		{"explicit false", boolPtr(false), "false"},
		{"explicit true", boolPtr(true), "true"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, sent, calls := secretsServer(t, infisical.AuthStrategy.UNIVERSAL_MACHINE_IDENTITY, true)

			resp := readSecrets(t, client, tc.config)

			if resp.Diagnostics.HasError() {
				t.Fatalf("a successful read must not error, got %v", resp.Diagnostics.Errors())
			}
			if calls.Load() != 1 {
				t.Fatalf("expected one request, got %d", calls.Load())
			}
			if *sent != tc.want {
				t.Errorf("expected include_imports=%s, got %q", tc.want, *sent)
			}
		})
	}
}

func TestSecretsDataSourceReadRejectsIncludeImportsWithServiceToken(t *testing.T) {
	client, _, calls := secretsServer(t, infisical.AuthStrategy.SERVICE_TOKEN, false)

	resp := readSecrets(t, client, boolPtr(true))

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected include_imports=true with a service token to fail")
	}
	if calls.Load() != 0 {
		t.Errorf("expected the failure before any request, got %d requests", calls.Load())
	}
}
