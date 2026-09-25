package resource

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	infisical "terraform-provider-infisical/internal/client"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// splunkCredentials builds a credentials object for the Splunk resource's schema.
func splunkCredentials(hostname, token string, port attr.Value) types.Object {
	return types.ObjectValueMust(map[string]attr.Type{
		"hostname": types.StringType,
		"port":     types.Int64Type,
		"token":    types.StringType,
	}, map[string]attr.Value{
		"hostname": types.StringValue(hostname),
		"port":     port,
		"token":    types.StringValue(token),
	})
}

func productFilter(products ...string) types.Object {
	values := make([]attr.Value, 0, len(products))
	for _, product := range products {
		values = append(values, types.StringValue(product))
	}
	return types.ObjectValueMust(auditLogStreamFilterAttrTypes, map[string]attr.Value{
		"products": types.SetValueMust(types.StringType, values),
	})
}

// fakeAuditLogStreamServer serves the stream API, returning credentials sanitized the way the
// real API does: the HEC token is never echoed back.
type fakeAuditLogStreamServer struct {
	hostname string
	port     any
	products []string
	exists   bool
	deleted  bool
}

func (f *fakeAuditLogStreamServer) start(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPatch:
			body := struct {
				Credentials struct {
					Hostname string `json:"hostname"`
					Port     *int   `json:"port"`
					Token    string `json:"token"`
				} `json:"credentials"`
				Filters *infisical.AuditLogStreamFilters `json:"filters"`
			}{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if body.Credentials.Token == "" {
				t.Errorf("%s sent no token", r.Method)
			}
			f.hostname = body.Credentials.Hostname
			f.port = nil
			if body.Credentials.Port != nil {
				f.port = *body.Credentials.Port
			}
			f.products = nil
			if body.Filters != nil {
				f.products = body.Filters.Products
			}
			f.exists = true
		case http.MethodGet:
			if !f.exists {
				http.NotFound(w, r)
				return
			}
		case http.MethodDelete:
			f.exists = false
			f.deleted = true
		}

		credentials := map[string]any{"hostname": f.hostname}
		if f.port != nil {
			credentials["port"] = f.port
		}
		response := map[string]any{
			"id": "stream-1", "orgId": "org-1", "provider": "splunk",
			"streamMode": "batch", "credentials": credentials,
		}
		if f.products != nil {
			response["filters"] = map[string]any{"products": f.products}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"auditLogStream": response})
	}))
	t.Cleanup(server.Close)
	return server
}

// baseResource unwraps a provider constructor so tests can inject a client.
func baseResource(t *testing.T, constructor func() resource.Resource) *AuditLogStreamBaseResource {
	t.Helper()
	r, ok := constructor().(*AuditLogStreamBaseResource)
	if !ok {
		t.Fatalf("resource is not an *AuditLogStreamBaseResource")
	}
	return r
}

func splunkResourceFor(t *testing.T, url string) (*AuditLogStreamBaseResource, resourceschema.Schema) {
	t.Helper()
	r := baseResource(t, NewAuditLogStreamSplunkResource)
	r.client = &infisical.Client{Config: infisical.Config{
		HttpClient:            resty.New().SetBaseURL(url),
		IsMachineIdentityAuth: true,
	}}

	var schemaResponse resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResponse)
	return r, schemaResponse.Schema
}

func TestAuditLogStreamLifecycle(t *testing.T) {
	ctx := context.Background()
	api := &fakeAuditLogStreamServer{}
	r, resourceSchema := splunkResourceFor(t, api.start(t).URL)

	planFor := func(credentials, filters types.Object) tfsdk.Plan {
		plan := tfsdk.Plan{Schema: resourceSchema}
		diags := plan.Set(ctx, &AuditLogStreamResourceModel{
			ID:          types.StringUnknown(),
			StreamMode:  types.StringUnknown(),
			Credentials: credentials,
			Filters:     filters,
		})
		if diags.HasError() {
			t.Fatal(diags)
		}
		return plan
	}

	createResponse := resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema}}
	r.Create(ctx, resource.CreateRequest{
		Plan: planFor(splunkCredentials("splunk.example.com", "hec-token", types.Int64Value(443)), productFilter("organization")),
	}, &createResponse)
	if createResponse.Diagnostics.HasError() {
		t.Fatal(createResponse.Diagnostics)
	}
	created := modelFrom(t, ctx, createResponse.State)
	if created.ID.ValueString() != "stream-1" || created.StreamMode.ValueString() != "batch" {
		t.Errorf("created = %#v", created)
	}
	assertCredential(t, created, "hostname", types.StringValue("splunk.example.com"))
	assertCredential(t, created, "port", types.Int64Value(443))
	assertCredential(t, created, "token", types.StringValue("hec-token"))

	// Read must keep the token, which the API never returns, while refreshing readable fields.
	readResponse := resource.ReadResponse{State: createResponse.State}
	r.Read(ctx, resource.ReadRequest{State: createResponse.State}, &readResponse)
	if readResponse.Diagnostics.HasError() {
		t.Fatal(readResponse.Diagnostics)
	}
	read := modelFrom(t, ctx, readResponse.State)
	assertCredential(t, read, "token", types.StringValue("hec-token"))
	if !read.Filters.Equal(productFilter("organization")) {
		t.Errorf("filters = %#v", read.Filters)
	}

	updateResponse := resource.UpdateResponse{State: readResponse.State}
	r.Update(ctx, resource.UpdateRequest{
		Plan:  planFor(splunkCredentials("splunk2.example.com", "hec-token-2", types.Int64Value(8088)), types.ObjectNull(auditLogStreamFilterAttrTypes)),
		State: readResponse.State,
	}, &updateResponse)
	if updateResponse.Diagnostics.HasError() {
		t.Fatal(updateResponse.Diagnostics)
	}
	updated := modelFrom(t, ctx, updateResponse.State)
	assertCredential(t, updated, "hostname", types.StringValue("splunk2.example.com"))
	assertCredential(t, updated, "port", types.Int64Value(8088))
	if api.products != nil {
		t.Errorf("products = %#v, want cleared", api.products)
	}

	deleteResponse := resource.DeleteResponse{State: updateResponse.State}
	r.Delete(ctx, resource.DeleteRequest{State: updateResponse.State}, &deleteResponse)
	if deleteResponse.Diagnostics.HasError() || !api.deleted {
		t.Fatalf("Delete() diagnostics = %v, deleted = %t", deleteResponse.Diagnostics, api.deleted)
	}
}

func TestAuditLogStreamReadRemovesDeletedStream(t *testing.T) {
	ctx := context.Background()
	api := &fakeAuditLogStreamServer{}
	r, resourceSchema := splunkResourceFor(t, api.start(t).URL)

	state := tfsdk.State{Schema: resourceSchema}
	if diags := state.Set(ctx, &AuditLogStreamResourceModel{
		ID:          types.StringValue("stream-1"),
		StreamMode:  types.StringValue("batch"),
		Credentials: splunkCredentials("splunk.example.com", "hec-token", types.Int64Null()),
		Filters:     types.ObjectNull(auditLogStreamFilterAttrTypes),
	}); diags.HasError() {
		t.Fatal(diags)
	}

	readResponse := resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, &readResponse)
	if readResponse.Diagnostics.HasError() {
		t.Fatal(readResponse.Diagnostics)
	}
	if !readResponse.State.Raw.IsNull() {
		t.Error("state was not removed for a stream the API no longer has")
	}
}

func TestAuditLogStreamRequiresMachineIdentity(t *testing.T) {
	r := baseResource(t, NewAuditLogStreamSplunkResource)
	r.client = &infisical.Client{Config: infisical.Config{IsMachineIdentityAuth: false}}

	createResponse := resource.CreateResponse{}
	r.Create(context.Background(), resource.CreateRequest{}, &createResponse)
	if !createResponse.Diagnostics.HasError() {
		t.Error("expected an error without machine identity auth")
	}
}

func TestAuditLogStreamRejectsProviderMismatch(t *testing.T) {
	r := baseResource(t, NewAuditLogStreamSplunkResource)
	model := AuditLogStreamResourceModel{}
	diags := r.applyStreamToState(&model, infisical.AuditLogStream{ID: "stream-1", Provider: "datadog"}, false)
	if !diags.HasError() {
		t.Error("expected an error when the API returns another provider's stream")
	}
}

// customCredentials builds a credentials object for the custom resource's schema.
func customCredentials(url string, headers map[string]string) types.Object {
	elements := make(map[string]attr.Value, len(headers))
	for key, value := range headers {
		elements[key] = types.StringValue(value)
	}
	return types.ObjectValueMust(map[string]attr.Type{
		"url":     types.StringType,
		"headers": types.MapType{ElemType: types.StringType},
	}, map[string]attr.Value{
		"url":     types.StringValue(url),
		"headers": types.MapValueMust(types.StringType, elements),
	})
}

// sumoLogicCredentials builds a credentials object for the Sumo Logic resource's schema.
func sumoLogicCredentials(url, token string) types.Object {
	return types.ObjectValueMust(map[string]attr.Type{
		"url":   types.StringType,
		"token": types.StringType,
	}, map[string]attr.Value{
		"url":   types.StringValue(url),
		"token": types.StringValue(token),
	})
}

func TestAuditLogStreamCustomHeadersRequest(t *testing.T) {
	r := baseResource(t, NewAuditLogStreamCustomResource)
	credentials, diags := r.credentialsToAPI(
		customCredentials("https://logs.example.com/ingest", map[string]string{"Authorization": "Bearer token"}),
		types.ObjectNull(r.credentialAttrTypes()),
	)
	if diags.HasError() {
		t.Fatal(diags)
	}

	headers, ok := credentials["headers"].([]map[string]string)
	if !ok || len(headers) != 1 {
		t.Fatalf("headers = %#v", credentials["headers"])
	}
	if headers[0]["key"] != "Authorization" || headers[0]["value"] != "Bearer token" {
		t.Errorf("header = %#v", headers[0])
	}
}

// Unset optional credentials are omitted so the API keeps the value it already holds.
func TestAuditLogStreamOmitsUnsetCredentials(t *testing.T) {
	r := baseResource(t, NewAuditLogStreamSplunkResource)
	credentials, diags := r.credentialsToAPI(
		splunkCredentials("splunk.example.com", "hec-token", types.Int64Null()),
		types.ObjectNull(r.credentialAttrTypes()),
	)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if _, present := credentials["port"]; present {
		t.Errorf("credentials = %#v, want no port", credentials)
	}
}

func TestAuditLogStreamFiltersToAPI(t *testing.T) {
	ctx := context.Background()

	if filters, diags := auditLogStreamFiltersToAPI(ctx, types.ObjectNull(auditLogStreamFilterAttrTypes)); diags.HasError() || filters != nil {
		t.Errorf("filters = %#v, %v; want nil", filters, diags)
	}

	filters, diags := auditLogStreamFiltersToAPI(ctx, productFilter("kms", "pam"))
	if diags.HasError() {
		t.Fatal(diags)
	}
	if filters == nil || len(filters.Products) != 2 {
		t.Fatalf("filters = %#v", filters)
	}
}

func modelFrom(t *testing.T, ctx context.Context, state tfsdk.State) AuditLogStreamResourceModel {
	t.Helper()
	var model AuditLogStreamResourceModel
	if diags := state.Get(ctx, &model); diags.HasError() {
		t.Fatal(diags)
	}
	return model
}

func assertCredential(t *testing.T, model AuditLogStreamResourceModel, name string, want attr.Value) {
	t.Helper()
	got := model.Credentials.Attributes()[name]
	if got == nil || !got.Equal(want) {
		t.Errorf("credentials[%q] = %#v, want %#v", name, got, want)
	}
}

// A stream that was created remotely must be recorded even when reconciling the response fails,
// otherwise Terraform forgets it and the next apply creates a duplicate.
func TestAuditLogStreamCreateRecordsStreamWhenReconcileFails(t *testing.T) {
	ctx := context.Background()
	api := &fakeAuditLogStreamServer{}
	r, resourceSchema := splunkResourceFor(t, api.start(t).URL)
	// The fake always answers with provider "splunk", so this forces the mismatch error after a
	// successful create.
	r.Provider = "datadog"

	plan := tfsdk.Plan{Schema: resourceSchema}
	if diags := plan.Set(ctx, &AuditLogStreamResourceModel{
		ID:          types.StringUnknown(),
		StreamMode:  types.StringUnknown(),
		Credentials: splunkCredentials("splunk.example.com", "hec-token", types.Int64Unknown()),
		Filters:     types.ObjectNull(auditLogStreamFilterAttrTypes),
	}); diags.HasError() {
		t.Fatal(diags)
	}

	createResponse := resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &createResponse)

	if !createResponse.Diagnostics.HasError() {
		t.Fatal("expected an error when the API returns another provider's stream")
	}
	if !api.exists {
		t.Fatal("expected the stream to have been created remotely")
	}
	if createResponse.State.Raw.IsNull() {
		t.Fatal("state was not written, so the created stream is untracked")
	}

	created := modelFrom(t, ctx, createResponse.State)
	if created.ID.ValueString() != "stream-1" {
		t.Errorf("id = %q, want the created stream ID", created.ID.ValueString())
	}
	// Unknowns must not survive into state.
	if created.StreamMode.IsUnknown() || created.Credentials.IsUnknown() {
		t.Errorf("state holds unknown values: stream_mode = %s, credentials = %s", created.StreamMode, created.Credentials)
	}
	if port := created.Credentials.Attributes()["port"]; port == nil || port.IsUnknown() {
		t.Errorf("credentials.port = %v, want a known value", port)
	}
}

func TestAuditLogStreamHeaderMapFromAPI(t *testing.T) {
	field := credentialField{Name: "headers", JSONName: "headers", Kind: credentialHeaderMap}

	value, ok := field.valueFromAPI([]any{
		map[string]any{"key": "Authorization", "value": "******"},
		map[string]any{"key": "X-Tenant", "value": "acme"},
	})
	if !ok {
		t.Fatal("valueFromAPI did not convert a header list")
	}
	want := types.MapValueMust(types.StringType, map[string]attr.Value{
		"Authorization": types.StringValue("******"),
		"X-Tenant":      types.StringValue("acme"),
	})
	if !value.Equal(want) {
		t.Errorf("valueFromAPI() = %s, want %s", value, want)
	}

	if _, ok := field.valueFromAPI("not-a-list"); ok {
		t.Error("expected a malformed header payload to be rejected")
	}
	if _, ok := field.valueFromAPI([]any{map[string]any{"key": "A"}}); ok {
		t.Error("expected a header with no value to be rejected")
	}
}

// An update that leaves a Sumo Logic token alone must not resend it, so a token rotated in
// Infisical is not overwritten.
func TestAuditLogStreamSumoLogicMasksUnchangedToken(t *testing.T) {
	r := baseResource(t, NewAuditLogStreamSumoLogicResource)
	state := sumoLogicCredentials("https://endpoint4.collection.sumologic.com/receiver/v1/http", "tok")

	unchanged, diags := r.credentialsToAPI(state, state)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if unchanged["token"] != infisical.AuditLogStreamRedactedCredential {
		t.Errorf("token = %#v, want the redaction sentinel", unchanged["token"])
	}

	// A deliberate rotation still has to reach the API.
	rotated, diags := r.credentialsToAPI(sumoLogicCredentials("https://endpoint4.collection.sumologic.com/receiver/v1/http", "tok-2"), state)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if rotated["token"] != "tok-2" {
		t.Errorf("token = %#v, want the new value", rotated["token"])
	}

	// On create there is no prior state, so the real token is sent.
	created, diags := r.credentialsToAPI(state, types.ObjectNull(r.credentialAttrTypes()))
	if diags.HasError() {
		t.Fatal(diags)
	}
	if created["token"] != "tok" {
		t.Errorf("token = %#v, want the real value", created["token"])
	}
}

// Custom headers are masked per key: changing one must not resend the others.
func TestAuditLogStreamCustomMasksUnchangedHeaders(t *testing.T) {
	r := baseResource(t, NewAuditLogStreamCustomResource)
	state := customCredentials("https://logs.example.com/ingest", map[string]string{
		"Authorization": "Bearer token",
		"X-Tenant":      "acme",
	})
	plan := customCredentials("https://logs.example.com/ingest", map[string]string{
		"Authorization": "Bearer token",
		"X-Tenant":      "globex",
	})

	credentials, diags := r.credentialsToAPI(plan, state)
	if diags.HasError() {
		t.Fatal(diags)
	}

	headers, ok := credentials["headers"].([]map[string]string)
	if !ok || len(headers) != 2 {
		t.Fatalf("headers = %#v", credentials["headers"])
	}
	got := map[string]string{}
	for _, header := range headers {
		got[header["key"]] = header["value"]
	}
	if got["Authorization"] != infisical.AuditLogStreamRedactedCredential {
		t.Errorf("Authorization = %q, want the redaction sentinel", got["Authorization"])
	}
	if got["X-Tenant"] != "globex" {
		t.Errorf("X-Tenant = %q, want the new value", got["X-Tenant"])
	}
}

// The providers whose API rejects the sentinel must keep sending the real secret.
func TestAuditLogStreamSendsSecretWhereMaskUnsupported(t *testing.T) {
	r := baseResource(t, NewAuditLogStreamSplunkResource)
	state := splunkCredentials("splunk.example.com", "hec-token", types.Int64Value(443))

	credentials, diags := r.credentialsToAPI(state, state)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if credentials["token"] != "hec-token" {
		t.Errorf("token = %#v, want the real value", credentials["token"])
	}
}
