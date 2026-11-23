package resource

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	infisical "terraform-provider-infisical/internal/client"
)

// TestIdentityAuthImport exercises import and its subsequent refresh against a read-only API.
func TestIdentityAuthImport(t *testing.T) {
	ctx := context.Background()
	zero := 0
	for _, method := range []string{"gcp", "oidc"} {
		variants := []string{"standalone", "template", "empty-trust"}
		if method == "gcp" {
			variants = []string{"gce", "iam"}
		}
		for _, variant := range variants {
			t.Run(method+"/"+variant, func(t *testing.T) {
				config := map[string]any{
					"id": "auth-config-id", "identityId": "identity-id",
					"accessTokenTTL": int64(1200), "accessTokenMaxTTL": int64(3600), "accessTokenNumUsesLimit": int64(7),
					"accessTokenTrustedIps": []infisical.IdentityAuthTrustedIp{
						{IpAddress: "0.0.0.0", Prefix: &zero}, {IpAddress: "::", Prefix: &zero}, {IpAddress: "192.0.2.1"},
					},
				}
				expected := map[string]any{
					"id": "auth-config-id", "identity_id": "identity-id",
					"access_token_ttl": int64(1200), "access_token_max_ttl": int64(3600), "access_token_num_uses_limit": int64(7),
					"access_token_trusted_ips": []IdentityGcpAuthResourceTrustedIps{
						{IpAddress: types.StringValue("0.0.0.0/0")}, {IpAddress: types.StringValue("::/0")}, {IpAddress: types.StringValue("192.0.2.1")},
					},
				}
				if method == "gcp" {
					config["type"] = variant
					config["allowedServiceAccounts"] = "one@example.com, two@example.com"
					config["allowedProjects"] = "project-one, project-two"
					config["allowedZones"] = "us-central1-a, us-central1-b"
					expected["type"] = variant
					expected["allowed_service_account_emails"] = []string{"one@example.com", "two@example.com"}
					expected["allowed_projects"] = []string{"project-one", "project-two"}
					expected["allowed_zones"] = []string{"us-central1-a", "us-central1-b"}
				} else {
					config["oidcDiscoveryUrl"] = "https://issuer.example.com"
					config["boundIssuer"] = "https://issuer.example.com"
					config["boundAudiences"] = "audience-one, audience-two"
					config["boundSubject"] = "repo:example/service:ref:refs/heads/main"
					config["boundClaims"] = map[string]string{"repository": "example/service"}
					config["claimMetadataMapping"] = map[string]string{"role": "token.groups"}
					config["caCert"] = "certificate"
					expected["oidc_discovery_url"] = "https://issuer.example.com"
					expected["bound_issuer"] = "https://issuer.example.com"
					expected["bound_audiences"] = []string{"audience-one", "audience-two"}
					expected["bound_subject"] = "repo:example/service:ref:refs/heads/main"
					expected["bound_claims"] = map[string]string{"repository": "example/service"}
					expected["claim_metadata_mapping"] = map[string]string{"role": "token.groups"}
					expected["oidc_ca_certificate"] = "certificate"
					expected["template_id"] = types.StringNull()
					if variant == "template" {
						config["templateId"] = "template-id"
						expected["template_id"] = types.StringValue("template-id")
					}
					if variant == "empty-trust" {
						config["boundAudiences"] = ""
						config["boundSubject"] = ""
						config["boundClaims"] = map[string]string{"": ""}
						config["claimMetadataMapping"] = map[string]string{}
						config["accessTokenNumUsesLimit"] = int64(0)
						expected["bound_audiences"] = []string{}
						expected["bound_subject"] = ""
						expected["bound_claims"] = map[string]string{"": ""}
						expected["claim_metadata_mapping"] = map[string]string{}
						expected["access_token_num_uses_limit"] = int64(0)
					}
				}
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					calls++
					if req.Method != http.MethodGet || req.URL.Path != "/api/v1/auth/"+method+"-auth/identities/identity-id" {
						t.Errorf("unexpected API request: %s %s", req.Method, req.URL.Path)
						http.Error(w, "unexpected request", http.StatusMethodNotAllowed)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					key := "identityGcpAuth"
					if method == "oidc" {
						key = "identityOidcAuth"
					}
					if err := json.NewEncoder(w).Encode(map[string]any{key: config}); err != nil {
						t.Error(err)
					}
				}))
				defer server.Close()
				r := identityAuthImportTestResource(method, server.URL, true)
				schemaResp := resource.SchemaResponse{}
				r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
				resp := resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
				r.(resource.ResourceWithImportState).ImportState(ctx, resource.ImportStateRequest{ID: "identity-id"}, &resp)
				if resp.Diagnostics.HasError() {
					t.Fatal(resp.Diagnostics)
				}
				assertIdentityAuthImportAttributes(t, resp.State, expected)
				readResp := resource.ReadResponse{State: resp.State}
				r.Read(ctx, resource.ReadRequest{State: resp.State}, &readResp)
				if readResp.Diagnostics.HasError() {
					t.Fatal(readResp.Diagnostics)
				}
				assertIdentityAuthImportAttributes(t, readResp.State, expected)
				if calls != 2 {
					t.Errorf("expected import and refresh to each GET once, got %d requests", calls)
				}
			})
		}
	}
}

// TestIdentityAuthImportErrors ensures unsupported auth and API failures never produce imported state.
func TestIdentityAuthImportErrors(t *testing.T) {
	for _, method := range []string{"gcp", "oidc"} {
		for _, status := range []int{http.StatusNotFound, http.StatusForbidden, http.StatusInternalServerError, 0} {
			t.Run(method+"/"+http.StatusText(status), func(t *testing.T) {
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					calls++
					if req.Method != http.MethodGet {
						t.Errorf("import must not write: %s", req.Method)
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(status)
					_, _ = w.Write([]byte(`{"message":"test API error"}`))
				}))
				defer server.Close()
				r := identityAuthImportTestResource(method, server.URL, status != 0)
				schemaResp := resource.SchemaResponse{}
				ctx := context.Background()
				r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
				resp := resource.ImportStateResponse{State: tfsdk.State{
					Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
				}}
				r.(resource.ResourceWithImportState).ImportState(ctx, resource.ImportStateRequest{ID: "identity-id"}, &resp)
				if !resp.Diagnostics.HasError() {
					t.Fatal("expected import diagnostic")
				}
				if !resp.State.Raw.IsNull() {
					t.Fatal("failed import must not populate state")
				}
				expectedCalls := 1
				if status == 0 {
					expectedCalls = 0
				}
				if calls != expectedCalls {
					t.Errorf("got %d requests, want %d", calls, expectedCalls)
				}
			})
		}
	}
}

// identityAuthImportTestResource configures a resource without authenticating against a live server.
func identityAuthImportTestResource(method, url string, machineAuth bool) resource.Resource {
	client := &infisical.Client{Config: infisical.Config{
		IsMachineIdentityAuth: machineAuth, HttpClient: resty.New().SetBaseURL(url),
	}}
	if method == "gcp" {
		return &IdentityGcpAuthResource{client: client}
	}
	return &IdentityOidcAuthResource{client: client}
}

// assertIdentityAuthImportAttributes checks API values survive both import and refresh.
func assertIdentityAuthImportAttributes(t *testing.T, state tfsdk.State, expected map[string]any) {
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
