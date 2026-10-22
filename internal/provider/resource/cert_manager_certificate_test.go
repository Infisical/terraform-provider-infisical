package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const (
	testCertNotBefore = "2026-10-01T00:00:00Z"
	testCertNotAfter  = "2026-10-31T00:00:00Z"
	testCSR           = "-----BEGIN CERTIFICATE REQUEST-----\nOLD\n-----END CERTIFICATE REQUEST-----"
	testRotatedCSR    = "-----BEGIN CERTIFICATE REQUEST-----\nNEW\n-----END CERTIFICATE REQUEST-----"
)

type certRequestLog struct {
	mu       sync.Mutex
	requests []string
}

func (l *certRequestLog) add(entry string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.requests = append(l.requests, entry)
}

func (l *certRequestLog) all() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.requests...)
}

func setTestNow(t *testing.T, now string) {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, now)
	if err != nil {
		t.Fatal(err)
	}
	previous := nowFunc
	nowFunc = func() time.Time { return parsed }
	t.Cleanup(func() { nowFunc = previous })
}

func certTestSchema(t *testing.T) schema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	NewCertManagerCertificateResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func newCertTestResource(t *testing.T, handler http.HandlerFunc) *certManagerCertificateResource {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &certManagerCertificateResource{client: &infisical.Client{Config: infisical.Config{
		HttpClient:            resty.New().SetBaseURL(server.URL),
		IsMachineIdentityAuth: true,
	}}}
}

func failOnRequest(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		http.NotFound(w, req)
	}
}

func fieldCertModel() certManagerCertificateResourceModel {
	return certManagerCertificateResourceModel{
		ProfileId:                   types.StringValue("profile-1"),
		ApplicationId:               types.StringValue("app-1"),
		CSR:                         types.StringNull(),
		CommonName:                  types.StringValue("api.example.com"),
		AltNames:                    types.ListNull(types.StringType),
		Organization:                types.StringNull(),
		OU:                          types.StringNull(),
		Country:                     types.StringNull(),
		Province:                    types.StringNull(),
		Locality:                    types.StringNull(),
		DomainComponents:            types.ListNull(types.StringType),
		KeyAlgorithm:                types.StringValue("RSA_2048"),
		SignatureAlgorithm:          types.StringValue("RSA-SHA256"),
		KeyUsages:                   types.ListNull(types.StringType),
		ExtendedKeyUsages:           types.ListNull(types.StringType),
		TTL:                         types.StringValue("30d"),
		TimeoutSeconds:              types.Int64Value(3600),
		ManualRenewDaysBeforeExpiry: types.Int64Null(),
		RevokeOnDestroy:             types.BoolValue(false),
		RevocationReason:            types.StringValue("unspecified"),
		DeleteOnDestroy:             types.BoolValue(false),
		Id:                          types.StringValue("cert-old"),
		OrderId:                     types.StringValue("order-1"),
		CertificateRequestId:        types.StringValue("req-old"),
		Status:                      types.StringValue("active"),
		SerialNumber:                types.StringValue("serial-old"),
		NotBefore:                   types.StringValue(testCertNotBefore),
		NotAfter:                    types.StringValue(testCertNotAfter),
		Certificate:                 types.StringValue("CERT-OLD"),
		PrivateKey:                  types.StringValue("KEY-OLD"),
		CertificateChain:            types.StringValue("CHAIN-OLD"),
	}
}

func csrCertModel() certManagerCertificateResourceModel {
	model := fieldCertModel()
	model.CSR = types.StringValue(testCSR)
	model.PrivateKey = types.StringValue("")
	return model
}

func configFromModel(model certManagerCertificateResourceModel) certManagerCertificateResourceModel {
	model.Id = types.StringNull()
	model.OrderId = types.StringNull()
	model.CertificateRequestId = types.StringNull()
	model.Status = types.StringNull()
	model.SerialNumber = types.StringNull()
	model.NotBefore = types.StringNull()
	model.NotAfter = types.StringNull()
	model.Certificate = types.StringNull()
	model.PrivateKey = types.StringNull()
	model.CertificateChain = types.StringNull()
	if !model.CSR.IsNull() {
		model.CommonName = types.StringNull()
		model.KeyAlgorithm = types.StringNull()
		model.SignatureAlgorithm = types.StringNull()
	}
	return model
}

func certState(t *testing.T, s schema.Schema, model *certManagerCertificateResourceModel) tfsdk.State {
	t.Helper()
	state := tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(context.Background()), nil)}
	if model != nil {
		if diags := state.Set(context.Background(), model); diags.HasError() {
			t.Fatal(diags)
		}
	}
	return state
}

func certPlan(t *testing.T, s schema.Schema, model *certManagerCertificateResourceModel) tfsdk.Plan {
	t.Helper()
	state := certState(t, s, model)
	return tfsdk.Plan{Schema: s, Raw: state.Raw}
}

func certConfig(t *testing.T, s schema.Schema, model *certManagerCertificateResourceModel) tfsdk.Config {
	t.Helper()
	state := certState(t, s, model)
	return tfsdk.Config{Schema: s, Raw: state.Raw}
}

func runModifyPlan(t *testing.T, plan, state *certManagerCertificateResourceModel) (resource.ModifyPlanResponse, resource.ModifyPlanRequest) {
	t.Helper()
	s := certTestSchema(t)
	var config *certManagerCertificateResourceModel
	if plan != nil {
		c := configFromModel(*plan)
		config = &c
	}
	req := resource.ModifyPlanRequest{
		Plan:   certPlan(t, s, plan),
		State:  certState(t, s, state),
		Config: certConfig(t, s, config),
	}
	resp := resource.ModifyPlanResponse{Plan: req.Plan}
	newCertTestResource(t, failOnRequest(t)).ModifyPlan(context.Background(), req, &resp)
	return resp, req
}

func planModel(t *testing.T, plan tfsdk.Plan) certManagerCertificateResourceModel {
	t.Helper()
	var model certManagerCertificateResourceModel
	if diags := plan.Get(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return model
}

func stateModel(t *testing.T, state tfsdk.State) certManagerCertificateResourceModel {
	t.Helper()
	var model certManagerCertificateResourceModel
	if diags := state.Get(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return model
}

func assertPlanUnchanged(t *testing.T, resp resource.ModifyPlanResponse, req resource.ModifyPlanRequest) {
	t.Helper()
	if !resp.Plan.Raw.Equal(req.Plan.Raw) {
		t.Fatal("plan was modified")
	}
	if len(resp.RequiresReplace) > 0 {
		t.Fatalf("unexpected replacement: %v", resp.RequiresReplace)
	}
}

func assertRenewalPlanned(t *testing.T, resp resource.ModifyPlanResponse) certManagerCertificateResourceModel {
	t.Helper()
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if len(resp.RequiresReplace) > 0 {
		t.Fatalf("renewal must be in place, got replacement of %v", resp.RequiresReplace)
	}
	planned := planModel(t, resp.Plan)
	for name, value := range map[string]types.String{
		"id": planned.Id, "certificate_request_id": planned.CertificateRequestId, "status": planned.Status,
		"serial_number": planned.SerialNumber, "not_before": planned.NotBefore, "not_after": planned.NotAfter,
		"certificate": planned.Certificate, "certificate_chain": planned.CertificateChain, "private_key": planned.PrivateKey,
	} {
		if !value.IsUnknown() {
			t.Errorf("%s must be unknown in a renewal plan, got %v", name, value)
		}
	}
	return planned
}

func hasWarning(resp resource.ModifyPlanResponse, summary string) bool {
	for _, d := range resp.Diagnostics.Warnings() {
		if d.Summary() == summary {
			return true
		}
	}
	return false
}

func TestCertModifyPlanCreateAndDestroyAreNoOps(t *testing.T) {
	setTestNow(t, "2026-10-29T00:00:00Z")
	model := fieldCertModel()
	model.ManualRenewDaysBeforeExpiry = types.Int64Value(10)

	resp, req := runModifyPlan(t, &model, nil)
	assertPlanUnchanged(t, resp, req)

	resp, req = runModifyPlan(t, nil, &model)
	assertPlanUnchanged(t, resp, req)
}

func TestCertModifyPlanRenewalWindow(t *testing.T) {
	tests := []struct {
		name        string
		now         string
		status      string
		renewDays   types.Int64
		wantRenewal bool
		wantWarning string
	}{
		{name: "outside_window", now: "2026-10-10T00:00:00Z", renewDays: types.Int64Value(10)},
		{name: "inside_window", now: "2026-10-25T00:00:00Z", renewDays: types.Int64Value(10), wantRenewal: true, wantWarning: "Certificate will be renewed"},
		{name: "window_boundary", now: "2026-10-21T00:00:00Z", renewDays: types.Int64Value(10), wantRenewal: true, wantWarning: "Certificate will be renewed"},
		{name: "expired", now: "2026-11-02T00:00:00Z", renewDays: types.Int64Value(10), wantWarning: "Certificate can no longer be renewed"},
		{name: "revoked", now: "2026-10-25T00:00:00Z", status: "revoked", renewDays: types.Int64Value(10), wantWarning: "Certificate can no longer be renewed"},
		{name: "renewal_gone", now: "2026-10-25T00:00:00Z", status: "renewed", renewDays: types.Int64Value(10), wantWarning: "Certificate can no longer be renewed"},
		{name: "attribute_not_set", now: "2026-10-30T00:00:00Z", renewDays: types.Int64Null()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setTestNow(t, tt.now)
			state := fieldCertModel()
			if tt.status != "" {
				state.Status = types.StringValue(tt.status)
			}
			plan := state
			plan.ManualRenewDaysBeforeExpiry = tt.renewDays
			state.ManualRenewDaysBeforeExpiry = tt.renewDays

			resp, req := runModifyPlan(t, &plan, &state)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			if tt.wantRenewal {
				planned := assertRenewalPlanned(t, resp)
				if planned.CommonName.ValueString() != "api.example.com" || planned.OrderId.ValueString() != "order-1" {
					t.Errorf("window renewal must keep subject and order, got %v %v", planned.CommonName, planned.OrderId)
				}
			} else {
				assertPlanUnchanged(t, resp, req)
			}
			if tt.wantWarning != "" && !hasWarning(resp, tt.wantWarning) {
				t.Errorf("missing warning %q, got %v", tt.wantWarning, resp.Diagnostics)
			}
			if tt.wantWarning == "" && resp.Diagnostics.WarningsCount() > 0 {
				t.Errorf("unexpected warnings: %v", resp.Diagnostics)
			}
		})
	}
}

func TestCertModifyPlanRejectsWindowNotSmallerThanLifetime(t *testing.T) {
	setTestNow(t, "2026-10-02T00:00:00Z")
	for _, tc := range []struct {
		stateTTL types.String
		planTTL  types.String
		days     int64
	}{
		{stateTTL: types.StringValue("30d"), planTTL: types.StringValue("30d"), days: 30},
		{stateTTL: types.StringValue("2m"), planTTL: types.StringValue("2m"), days: 31},
		{stateTTL: types.StringNull(), planTTL: types.StringNull(), days: 45},
		{stateTTL: types.StringValue("30d"), planTTL: types.StringValue("4w"), days: 28},
		{stateTTL: types.StringValue("30d"), planTTL: types.StringValue("90m"), days: 1},
		{stateTTL: types.StringValue("30d"), planTTL: types.StringValue("12h"), days: 1},
	} {
		state := fieldCertModel()
		state.TTL = tc.stateTTL
		plan := state
		plan.TTL = tc.planTTL
		plan.ManualRenewDaysBeforeExpiry = types.Int64Value(tc.days)

		resp, _ := runModifyPlan(t, &plan, &state)
		if !resp.Diagnostics.HasError() {
			t.Errorf("ttl %v -> %v with %d days: expected an error", tc.stateTTL, tc.planTTL, tc.days)
		}
	}
}

func TestCertModifyPlanAcceptsWindowSmallerThanRealLifetime(t *testing.T) {
	setTestNow(t, "2026-10-02T00:00:00Z")
	state := fieldCertModel()
	state.TTL = types.StringValue("720h")
	plan := state
	plan.ManualRenewDaysBeforeExpiry = types.Int64Value(29)

	resp, _ := runModifyPlan(t, &plan, &state)
	if resp.Diagnostics.HasError() {
		t.Fatalf("29 days is smaller than the 30 day certificate, got %v", resp.Diagnostics)
	}
}

func TestCertTTLParsingFollowsRequestUnits(t *testing.T) {
	for ttl, want := range map[string]float64{"1y": 365, "2w": 14, "30d": 30, "36h": 1.5, "1440m": 1, "86400s": 1} {
		got, ok := parseCertificateTTLDays(ttl)
		if !ok || got != want {
			t.Errorf("parseCertificateTTLDays(%q) = %v, %t, want %v", ttl, got, ok, want)
		}
	}
	for _, ttl := range []string{"", "d", "30", "-5d", "1x"} {
		if _, ok := parseCertificateTTLDays(ttl); ok {
			t.Errorf("parseCertificateTTLDays(%q) should not parse", ttl)
		}
	}
}

func TestCertModifyPlanCSRChangeRenewsInPlace(t *testing.T) {
	setTestNow(t, "2026-10-05T00:00:00Z")
	state := csrCertModel()
	plan := state
	plan.CSR = types.StringValue(testRotatedCSR)

	resp, _ := runModifyPlan(t, &plan, &state)
	planned := assertRenewalPlanned(t, resp)
	if !planned.CommonName.IsUnknown() || !planned.KeyAlgorithm.IsUnknown() || !planned.AltNames.IsUnknown() {
		t.Error("subject attributes come from the new CSR and must be unknown")
	}
	if planned.CSR.ValueString() != testRotatedCSR {
		t.Errorf("csr = %v, want rotated CSR", planned.CSR)
	}
}

func TestCertModifyPlanRemovingCSRReplaces(t *testing.T) {
	setTestNow(t, "2026-10-05T00:00:00Z")
	s := certTestSchema(t)
	state := csrCertModel()
	plan := state
	config := configFromModel(state)
	config.CSR = types.StringNull()
	config.CommonName = types.StringValue("api.example.com")
	req := resource.ModifyPlanRequest{
		Plan:   certPlan(t, s, &plan),
		State:  certState(t, s, &state),
		Config: certConfig(t, s, &config),
	}
	resp := resource.ModifyPlanResponse{Plan: req.Plan}
	newCertTestResource(t, failOnRequest(t)).ModifyPlan(context.Background(), req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if len(resp.RequiresReplace) != 1 || !resp.RequiresReplace[0].Equal(path.Root("csr")) {
		t.Fatalf("RequiresReplace = %v, want csr", resp.RequiresReplace)
	}
	if !planModel(t, resp.Plan).CSR.IsNull() {
		t.Error("csr must be planned as null so the replacement issues from fields")
	}
}

func TestCertModifyPlanCSRChangeOnRevokedCertificateFails(t *testing.T) {
	setTestNow(t, "2026-10-05T00:00:00Z")
	state := csrCertModel()
	state.Status = types.StringValue("revoked")
	plan := state
	plan.CSR = types.StringValue(testRotatedCSR)

	resp, _ := runModifyPlan(t, &plan, &state)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error for a CSR change on a revoked certificate")
	}
}

func TestCertCSRReplacementOnlyWhenIssuanceModeChanges(t *testing.T) {
	ctx := context.Background()
	s := certTestSchema(t)
	stateRaw := certState(t, s, ptr(csrCertModel())).Raw
	tests := []struct {
		name  string
		state types.String
		plan  types.String
		want  bool
	}{
		{name: "csr_rotated", state: types.StringValue(testCSR), plan: types.StringValue(testRotatedCSR), want: false},
		{name: "field_to_csr", state: types.StringNull(), plan: types.StringValue(testCSR), want: true},
		{name: "csr_to_empty", state: types.StringValue(testCSR), plan: types.StringValue(""), want: true},
		{name: "unknown_csr_on_csr_cert", state: types.StringValue(testCSR), plan: types.StringUnknown(), want: false},
		{name: "unknown_csr_on_field_cert", state: types.StringNull(), plan: types.StringUnknown(), want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := planmodifier.StringRequest{
				State:      tfsdk.State{Schema: s, Raw: stateRaw},
				StateValue: tt.state,
				PlanValue:  tt.plan,
			}
			var resp stringplanmodifier.RequiresReplaceIfFuncResponse
			certificateIssuanceModeChanged(ctx, req, &resp)
			if resp.RequiresReplace != tt.want {
				t.Errorf("RequiresReplace = %t, want %t", resp.RequiresReplace, tt.want)
			}
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}

func runUpdate(t *testing.T, r *certManagerCertificateResource, plan tfsdk.Plan, state certManagerCertificateResourceModel) (resource.UpdateResponse, certManagerCertificateResourceModel) {
	t.Helper()
	s := certTestSchema(t)
	priorState := certState(t, s, &state)
	resp := resource.UpdateResponse{State: priorState}
	r.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: priorState}, &resp)
	return resp, stateModel(t, resp.State)
}

func renewalPlan(t *testing.T, plan, state certManagerCertificateResourceModel) tfsdk.Plan {
	t.Helper()
	resp, _ := runModifyPlan(t, &plan, &state)
	assertRenewalPlanned(t, resp)
	return resp.Plan
}

func writeCertDetails(w http.ResponseWriter, id, serial string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, `{"certificate":{"id":%q,"status":"active","commonName":"api.example.com","serialNumber":%q,"notBefore":"2026-10-25T00:00:00Z","notAfter":"2026-11-24T00:00:00Z","keyAlgorithm":"RSA_2048","signatureAlgorithm":"RSA-SHA256","profileId":"profile-1","applicationId":"app-1","orderId":"order-1"}}`, id, serial)
}

func TestCertUpdateFlagOnlyChangeMakesNoRequests(t *testing.T) {
	setTestNow(t, "2026-10-05T00:00:00Z")
	state := csrCertModel()
	plan := state
	plan.RevokeOnDestroy = types.BoolValue(true)
	plan.RevocationReason = types.StringValue("superseded")
	plan.DeleteOnDestroy = types.BoolValue(true)
	plan.ManualRenewDaysBeforeExpiry = types.Int64Value(7)
	plan.TimeoutSeconds = types.Int64Value(60)
	plan.Status = types.StringUnknown()

	resp, got := runUpdate(t, newCertTestResource(t, failOnRequest(t)), certPlan(t, certTestSchema(t), &plan), state)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if !got.RevokeOnDestroy.ValueBool() || !got.DeleteOnDestroy.ValueBool() || got.RevocationReason.ValueString() != "superseded" ||
		got.ManualRenewDaysBeforeExpiry.ValueInt64() != 7 || got.TimeoutSeconds.ValueInt64() != 60 {
		t.Errorf("in-place fields not applied: %+v", got)
	}
	if got.Id.ValueString() != "cert-old" || got.Status.ValueString() != "active" || got.Certificate.ValueString() != "CERT-OLD" {
		t.Errorf("certificate fields must stay unchanged: %+v", got)
	}
}

func TestCertUpdateRenewsCSRCertificate(t *testing.T) {
	setTestNow(t, "2026-10-05T00:00:00Z")
	state := csrCertModel()
	plan := state
	plan.CSR = types.StringValue(strings.ReplaceAll(testRotatedCSR, "\n", "\r\n") + "\n")

	log := &certRequestLog{}
	r := newCertTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		log.add(req.Method + " " + req.URL.Path)
		switch req.Method + " " + req.URL.Path {
		case "POST /api/v1/cert-manager/certificates/cert-old/renew":
			var body map[string]interface{}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["renewalKeySource"] != "csr" {
				t.Errorf("renewalKeySource = %v, want csr", body["renewalKeySource"])
			}
			if body["csr"] != testRotatedCSR {
				t.Errorf("csr = %q, want the normalized new CSR", body["csr"])
			}
			if _, ok := body["attributes"]; ok {
				t.Errorf("renewals copy the original certificate and must not send attributes, got %v", body["attributes"])
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"certificate":"CERT-NEW","certificateChain":"CHAIN-NEW","issuingCaCertificate":"CA","serialNumber":"serial-new","certificateId":"cert-new","certificateRequestId":"req-new"}`)
		case "GET /api/v1/cert-manager/certificates/cert-new":
			writeCertDetails(w, "cert-new", "serial-new")
		default:
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			http.NotFound(w, req)
		}
	})

	resp, got := runUpdate(t, r, renewalPlan(t, plan, state), state)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if got.Id.ValueString() != "cert-new" || got.Certificate.ValueString() != "CERT-NEW" || got.SerialNumber.ValueString() != "serial-new" ||
		got.CertificateRequestId.ValueString() != "req-new" || got.Status.ValueString() != "active" || got.PrivateKey.ValueString() != "" {
		t.Errorf("unexpected state after renewal: %+v", got)
	}
	if strings.Join(log.all(), ",") != "POST /api/v1/cert-manager/certificates/cert-old/renew,GET /api/v1/cert-manager/certificates/cert-new" {
		t.Errorf("requests = %v", log.all())
	}
}

func TestCertUpdateRenewsFieldCertificateAndFetchesNewKey(t *testing.T) {
	setTestNow(t, "2026-10-25T00:00:00Z")
	state := fieldCertModel()
	state.ManualRenewDaysBeforeExpiry = types.Int64Value(10)
	plan := state

	r := newCertTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.Method + " " + req.URL.Path {
		case "POST /api/v1/cert-manager/certificates/cert-old/renew":
			var body map[string]interface{}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if _, ok := body["renewalKeySource"]; ok {
				t.Errorf("field certificates must not send a key source, got %v", body)
			}
			if _, ok := body["csr"]; ok {
				t.Errorf("field certificates must not send a CSR, got %v", body)
			}
			_, _ = fmt.Fprint(w, `{"certificate":"CERT-NEW","certificateChain":"CHAIN-NEW","issuingCaCertificate":"CA","serialNumber":"serial-new","certificateId":"cert-new","certificateRequestId":"req-new"}`)
		case "GET /api/v1/cert-manager/certificates/cert-new":
			writeCertDetails(w, "cert-new", "serial-new")
		case "GET /api/v1/cert-manager/certificates/cert-new/bundle":
			_, _ = fmt.Fprint(w, `{"certificate":"CERT-NEW","certificateChain":"CHAIN-NEW","privateKey":"KEY-NEW","serialNumber":"serial-new"}`)
		default:
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			http.NotFound(w, req)
		}
	})

	resp, got := runUpdate(t, r, renewalPlan(t, plan, state), state)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if got.Id.ValueString() != "cert-new" || got.PrivateKey.ValueString() != "KEY-NEW" || got.NotAfter.ValueString() != "2026-11-24T00:00:00Z" {
		t.Errorf("unexpected state after renewal: %+v", got)
	}
}

func TestCertUpdateAsyncRenewalIgnoresPlaceholderId(t *testing.T) {
	setTestNow(t, "2026-10-25T00:00:00Z")
	state := fieldCertModel()
	state.ManualRenewDaysBeforeExpiry = types.Int64Value(10)
	plan := state

	r := newCertTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.Method + " " + req.URL.Path {
		case "POST /api/v1/cert-manager/certificates/cert-old/renew":
			_, _ = fmt.Fprint(w, `{"certificate":"","certificateChain":"","issuingCaCertificate":"","serialNumber":"","certificateId":"placeholder","certificateRequestId":"req-async"}`)
		case "GET /api/v1/cert-manager/certificates/certificate-requests/req-async":
			_, _ = fmt.Fprint(w, `{"status":"issued","certificateId":"cert-real","certificate":"CERT-NEW","certificateChain":"CHAIN-NEW","serialNumber":"serial-new"}`)
		case "GET /api/v1/cert-manager/certificates/cert-real":
			writeCertDetails(w, "cert-real", "serial-new")
		case "GET /api/v1/cert-manager/certificates/cert-real/bundle":
			_, _ = fmt.Fprint(w, `{"certificate":"CERT-NEW","certificateChain":"CHAIN-NEW","privateKey":"KEY-NEW","serialNumber":"serial-new"}`)
		default:
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			http.NotFound(w, req)
		}
	})

	resp, got := runUpdate(t, r, renewalPlan(t, plan, state), state)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if got.Id.ValueString() != "cert-real" || got.CertificateRequestId.ValueString() != "req-async" || got.PrivateKey.ValueString() != "KEY-NEW" {
		t.Errorf("unexpected state after async renewal: %+v", got)
	}
}

func runDelete(t *testing.T, r *certManagerCertificateResource, state certManagerCertificateResourceModel) resource.DeleteResponse {
	t.Helper()
	s := certTestSchema(t)
	priorState := certState(t, s, &state)
	resp := resource.DeleteResponse{State: priorState}
	r.Delete(context.Background(), resource.DeleteRequest{State: priorState}, &resp)
	return resp
}

func deleteTestServer(t *testing.T, log *certRequestLog, status string, revokeStatus, deleteStatus int, wantReason string) *certManagerCertificateResource {
	return newCertTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		log.add(req.Method + " " + req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch req.Method + " " + req.URL.Path {
		case "GET /api/v1/cert-manager/certificates/cert-old":
			if req.URL.Query().Get("includeDeleted") != "true" {
				t.Errorf("destroy must read the certificate with includeDeleted, got %q", req.URL.RawQuery)
			}
			switch status {
			case "":
				http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
			case "self_signed":
				_, _ = fmt.Fprint(w, `{"certificate":{"id":"cert-old","status":"active"}}`)
			case "deleted":
				_, _ = fmt.Fprint(w, `{"certificate":{"id":"cert-old","status":"active","caId":"ca-1","deletedAt":"2026-10-10T00:00:00Z"}}`)
			default:
				_, _ = fmt.Fprintf(w, `{"certificate":{"id":"cert-old","status":%q,"caId":"ca-1"}}`, status)
			}
		case "POST /api/v1/cert-manager/certificates/cert-old/revoke":
			var body map[string]string
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["revocationReason"] != wantReason {
				t.Errorf("revocationReason = %q, want %q", body["revocationReason"], wantReason)
			}
			w.WriteHeader(revokeStatus)
			_, _ = fmt.Fprint(w, `{"message":"ok"}`)
		case "DELETE /api/v1/cert-manager/certificates/cert-old":
			w.WriteHeader(deleteStatus)
			_, _ = fmt.Fprint(w, `{}`)
		default:
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			http.NotFound(w, req)
		}
	})
}

func TestCertDelete(t *testing.T) {
	const (
		get    = "GET /api/v1/cert-manager/certificates/cert-old"
		revoke = "POST /api/v1/cert-manager/certificates/cert-old/revoke"
		del    = "DELETE /api/v1/cert-manager/certificates/cert-old"
	)
	tests := []struct {
		name         string
		now          string
		revoke       bool
		delete       bool
		reason       string
		serverStatus string
		revokeStatus int
		deleteStatus int
		wantReason   string
		wantRequests []string
		wantError    bool
	}{
		{name: "flags_off", now: "2026-10-05T00:00:00Z"},
		{name: "revoke_with_mapped_reason", now: "2026-10-05T00:00:00Z", revoke: true, reason: "key_compromise", serverStatus: "active", wantReason: "KEY_COMPROMISE", wantRequests: []string{get, revoke}},
		{name: "aa_compromise_mapping", now: "2026-10-05T00:00:00Z", revoke: true, reason: "aa_compromise", serverStatus: "active", wantReason: "A_A_COMPROMISE", wantRequests: []string{get, revoke}},
		{name: "expired_skips_revoke", now: "2026-11-05T00:00:00Z", revoke: true, reason: "unspecified", serverStatus: "expired", wantRequests: []string{get}},
		{name: "expired_still_deletes", now: "2026-11-05T00:00:00Z", delete: true, reason: "unspecified", serverStatus: "expired", wantRequests: []string{get, del}},
		{name: "already_revoked_skips_revoke", now: "2026-10-05T00:00:00Z", revoke: true, delete: true, reason: "unspecified", serverStatus: "revoked", wantRequests: []string{get, del}},
		{name: "delete_only_does_not_revoke", now: "2026-10-05T00:00:00Z", delete: true, reason: "superseded", serverStatus: "active", wantRequests: []string{get, del}},
		{name: "revoke_then_delete", now: "2026-10-05T00:00:00Z", revoke: true, delete: true, reason: "superseded", serverStatus: "active", wantReason: "SUPERSEDED", wantRequests: []string{get, revoke, del}},
		{name: "self_signed_skips_revoke_and_deletes", now: "2026-10-05T00:00:00Z", revoke: true, delete: true, reason: "unspecified", serverStatus: "self_signed", wantRequests: []string{get, del}},
		{name: "deleted_outside_terraform_is_still_revoked", now: "2026-10-05T00:00:00Z", revoke: true, delete: true, reason: "key_compromise", serverStatus: "deleted", wantReason: "KEY_COMPROMISE", wantRequests: []string{get, revoke}},
		{name: "certificate_gone", now: "2026-10-05T00:00:00Z", revoke: true, delete: true, reason: "unspecified", wantRequests: []string{get}},
		{name: "revoke_404", now: "2026-10-05T00:00:00Z", revoke: true, reason: "unspecified", serverStatus: "active", revokeStatus: http.StatusNotFound, wantReason: "UNSPECIFIED", wantRequests: []string{get, revoke}},
		{name: "delete_404", now: "2026-11-05T00:00:00Z", delete: true, reason: "unspecified", serverStatus: "expired", deleteStatus: http.StatusNotFound, wantRequests: []string{get, del}},
		{name: "revoke_error_fails", now: "2026-10-05T00:00:00Z", revoke: true, delete: true, reason: "unspecified", serverStatus: "active", revokeStatus: http.StatusBadRequest, wantReason: "UNSPECIFIED", wantRequests: []string{get, revoke}, wantError: true},
		{name: "delete_error_fails", now: "2026-10-05T00:00:00Z", revoke: true, delete: true, reason: "unspecified", serverStatus: "active", deleteStatus: http.StatusBadRequest, wantReason: "UNSPECIFIED", wantRequests: []string{get, revoke, del}, wantError: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setTestNow(t, tt.now)
			if tt.revokeStatus == 0 {
				tt.revokeStatus = http.StatusOK
			}
			if tt.deleteStatus == 0 {
				tt.deleteStatus = http.StatusOK
			}
			state := fieldCertModel()
			state.RevokeOnDestroy = types.BoolValue(tt.revoke)
			state.DeleteOnDestroy = types.BoolValue(tt.delete)
			if tt.reason != "" {
				state.RevocationReason = types.StringValue(tt.reason)
			}
			log := &certRequestLog{}
			resp := runDelete(t, deleteTestServer(t, log, tt.serverStatus, tt.revokeStatus, tt.deleteStatus, tt.wantReason), state)
			if resp.Diagnostics.HasError() != tt.wantError {
				t.Fatalf("error = %v, want error %t", resp.Diagnostics, tt.wantError)
			}
			if strings.Join(log.all(), ",") != strings.Join(tt.wantRequests, ",") {
				t.Errorf("requests = %v, want %v", log.all(), tt.wantRequests)
			}
		})
	}
}

func runRead(t *testing.T, r *certManagerCertificateResource, state certManagerCertificateResourceModel) resource.ReadResponse {
	t.Helper()
	s := certTestSchema(t)
	priorState := certState(t, s, &state)
	resp := resource.ReadResponse{State: priorState}
	r.Read(context.Background(), resource.ReadRequest{State: priorState}, &resp)
	return resp
}

func TestCertReadFollowsLatestRenewal(t *testing.T) {
	for _, bundleForbidden := range []bool{false, true} {
		t.Run(fmt.Sprintf("bundle_forbidden_%t", bundleForbidden), func(t *testing.T) {
			r := newCertTestResource(t, func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch req.Method + " " + req.URL.Path {
				case "GET /api/v1/cert-manager/certificates/cert-old":
					_, _ = fmt.Fprintf(w, `{"certificate":{"id":"cert-old","status":"active","commonName":"api.example.com","serialNumber":"serial-old","notBefore":%q,"notAfter":%q,"renewedByCertificateId":"cert-mid","latestRenewalCertificateId":"cert-latest","orderId":"order-1"}}`, testCertNotBefore, testCertNotAfter)
				case "GET /api/v1/cert-manager/certificates/cert-latest":
					writeCertDetails(w, "cert-latest", "serial-latest")
				case "GET /api/v1/cert-manager/certificates/cert-latest/bundle":
					if bundleForbidden {
						w.WriteHeader(http.StatusForbidden)
						_, _ = fmt.Fprint(w, `{"message":"forbidden"}`)
						return
					}
					_, _ = fmt.Fprint(w, `{"certificate":"CERT-LATEST","certificateChain":"CHAIN-LATEST","privateKey":"KEY-LATEST","serialNumber":"serial-latest"}`)
				case "GET /api/v1/cert-manager/certificates/cert-latest/certificate":
					_, _ = fmt.Fprint(w, `{"certificate":"CERT-LATEST","certificateChain":"CHAIN-LATEST","serialNumber":"serial-latest"}`)
				default:
					t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
					http.NotFound(w, req)
				}
			})

			state := fieldCertModel()
			state.RevokeOnDestroy = types.BoolValue(true)
			state.ManualRenewDaysBeforeExpiry = types.Int64Value(5)
			resp := runRead(t, r, state)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			got := stateModel(t, resp.State)
			wantKey := "KEY-LATEST"
			if bundleForbidden {
				wantKey = ""
				if resp.Diagnostics.WarningsCount() != 1 {
					t.Errorf("expected a warning about the private key, got %v", resp.Diagnostics)
				}
			}
			if got.Id.ValueString() != "cert-latest" || got.SerialNumber.ValueString() != "serial-latest" || got.Certificate.ValueString() != "CERT-LATEST" ||
				got.CertificateChain.ValueString() != "CHAIN-LATEST" || got.PrivateKey.ValueString() != wantKey || got.NotAfter.ValueString() != "2026-11-24T00:00:00Z" {
				t.Errorf("unexpected state after following renewal: %+v", got)
			}
			if !got.RevokeOnDestroy.ValueBool() || got.ManualRenewDaysBeforeExpiry.ValueInt64() != 5 {
				t.Errorf("Read must not overwrite lifecycle settings: %+v", got)
			}
		})
	}
}

func TestCertReadMarksCertificateWhoseRenewalIsGone(t *testing.T) {
	r := newCertTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"certificate":{"id":"cert-old","status":"active","commonName":"api.example.com","serialNumber":"serial-old","notBefore":%q,"notAfter":%q,"renewedByCertificateId":"cert-deleted","latestRenewalCertificateId":null,"orderId":"order-1"}}`, testCertNotBefore, testCertNotAfter)
	})
	resp := runRead(t, r, fieldCertModel())
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	got := stateModel(t, resp.State)
	if got.Id.ValueString() != "cert-old" || got.Status.ValueString() != certificateStatusRenewed {
		t.Errorf("expected cert-old with status renewed, got %v %v", got.Id, got.Status)
	}
}

func TestCertReadRemovesDeletedCertificate(t *testing.T) {
	r := newCertTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		http.Error(w, `{"message":"Certificate not found"}`, http.StatusNotFound)
	})
	resp := runRead(t, r, fieldCertModel())
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Error("deleted certificate must be removed from state")
	}
}

func TestCertUnconfiguredComputedAttributesDoNotForceReplacement(t *testing.T) {
	ctx := context.Background()
	s := certTestSchema(t)
	for _, model := range []certManagerCertificateResourceModel{csrCertModel(), fieldCertModel()} {
		priorState := certState(t, s, &model)
		for name, attribute := range s.Attributes {
			if !attribute.IsComputed() || attribute.IsRequired() {
				continue
			}
			switch typed := attribute.(type) {
			case schema.StringAttribute:
				var stateValue types.String
				priorState.GetAttribute(ctx, path.Root(name), &stateValue)
				req := planmodifier.StringRequest{Path: path.Root(name), State: priorState, Plan: tfsdk.Plan{Schema: s, Raw: priorState.Raw}, StateValue: stateValue, PlanValue: types.StringUnknown(), ConfigValue: types.StringNull()}
				for _, modifier := range typed.PlanModifiers {
					resp := planmodifier.StringResponse{PlanValue: req.PlanValue}
					modifier.PlanModifyString(ctx, req, &resp)
					if resp.RequiresReplace {
						t.Errorf("%s: unset %s forces replacement on an in-place change", model.Id.ValueString(), name)
					}
					req.PlanValue = resp.PlanValue
				}
			case schema.ListAttribute:
				var stateValue types.List
				priorState.GetAttribute(ctx, path.Root(name), &stateValue)
				req := planmodifier.ListRequest{Path: path.Root(name), State: priorState, Plan: tfsdk.Plan{Schema: s, Raw: priorState.Raw}, StateValue: stateValue, PlanValue: types.ListUnknown(types.StringType), ConfigValue: types.ListNull(types.StringType)}
				for _, modifier := range typed.PlanModifiers {
					resp := planmodifier.ListResponse{PlanValue: req.PlanValue}
					modifier.PlanModifyList(ctx, req, &resp)
					if resp.RequiresReplace {
						t.Errorf("unset %s forces replacement on an in-place change", name)
					}
					req.PlanValue = resp.PlanValue
				}
			}
		}
	}
}

func TestCertReadPopulatesMaterialAfterImport(t *testing.T) {
	r := newCertTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.Method + " " + req.URL.Path {
		case "GET /api/v1/cert-manager/certificates/cert-old":
			_, _ = fmt.Fprintf(w, `{"certificate":{"id":"cert-old","status":"active","commonName":"api.example.com","serialNumber":"serial-old","notBefore":%q,"notAfter":%q,"hasPrivateKey":true}}`, testCertNotBefore, testCertNotAfter)
		case "GET /api/v1/cert-manager/certificates/cert-old/bundle":
			_, _ = fmt.Fprint(w, `{"certificate":"CERT-OLD","certificateChain":"CHAIN-OLD","privateKey":"KEY-OLD","serialNumber":"serial-old"}`)
		default:
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			http.NotFound(w, req)
		}
	})

	imported := fieldCertModel()
	imported.Certificate = types.StringNull()
	imported.CertificateChain = types.StringNull()
	imported.PrivateKey = types.StringNull()
	resp := runRead(t, r, imported)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	got := stateModel(t, resp.State)
	if got.Certificate.ValueString() != "CERT-OLD" || got.CertificateChain.ValueString() != "CHAIN-OLD" || got.PrivateKey.ValueString() != "KEY-OLD" {
		t.Errorf("imported certificate material not populated: %+v", got)
	}
}

func renewTestServer(t *testing.T, wantBody func(map[string]interface{}), renewResponse string, bundleStatus int, log *certRequestLog) *certManagerCertificateResource {
	return renewTestServerWithCa(t, "internal", wantBody, renewResponse, bundleStatus, log)
}

func renewTestServerWithCa(t *testing.T, caType string, wantBody func(map[string]interface{}), renewResponse string, bundleStatus int, log *certRequestLog) *certManagerCertificateResource {
	return newCertTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		log.add(req.Method + " " + req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch req.Method + " " + req.URL.Path {
		case "GET /api/v1/cert-manager/certificates/cert-old":
			_, _ = fmt.Fprintf(w, `{"certificate":{"id":"cert-old","status":"active","caId":"ca-1","caType":%q}}`, caType)
		case "POST /api/v1/cert-manager/certificates/cert-old/renew":
			var body map[string]interface{}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			wantBody(body)
			_, _ = fmt.Fprint(w, renewResponse)
		case "GET /api/v1/cert-manager/certificates/cert-new":
			writeCertDetails(w, "cert-new", "serial-new")
		case "GET /api/v1/cert-manager/certificates/cert-new/bundle":
			w.WriteHeader(bundleStatus)
			_, _ = fmt.Fprint(w, `{"certificate":"CERT-NEW","certificateChain":"CHAIN-NEW","privateKey":"KEY-BUNDLE","serialNumber":"serial-new"}`)
		default:
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			http.NotFound(w, req)
		}
	})
}

const testRenewResponse = `{"certificate":"CERT-NEW","certificateChain":"CHAIN-NEW","issuingCaCertificate":"CA","serialNumber":"serial-new","certificateId":"cert-new","certificateRequestId":"req-new"}`

func TestCertUpdateWindowRenewalOfCSRCertificateReusesTheCSR(t *testing.T) {
	setTestNow(t, "2026-10-25T00:00:00Z")
	state := csrCertModel()
	state.ManualRenewDaysBeforeExpiry = types.Int64Value(10)
	log := &certRequestLog{}
	r := renewTestServer(t, func(body map[string]interface{}) {
		if body["renewalKeySource"] != "reuse" {
			t.Errorf("renewalKeySource = %v, want reuse so the renewal is not a certificate edit", body["renewalKeySource"])
		}
		if _, ok := body["csr"]; ok {
			t.Errorf("an unchanged CSR must not be sent, got %v", body)
		}
		if _, ok := body["attributes"]; ok {
			t.Errorf("attributes must not be sent, got %v", body)
		}
	}, testRenewResponse, http.StatusOK, log)

	resp, got := runUpdate(t, r, renewalPlan(t, state, state), state)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if got.Id.ValueString() != "cert-new" || got.PrivateKey.ValueString() != "" {
		t.Errorf("unexpected state after CSR window renewal: %+v", got)
	}
}

func TestCertUpdateFieldRenewalUsesKeyFromRenewResponse(t *testing.T) {
	setTestNow(t, "2026-10-25T00:00:00Z")
	state := fieldCertModel()
	state.ManualRenewDaysBeforeExpiry = types.Int64Value(10)
	log := &certRequestLog{}
	r := renewTestServer(t, func(map[string]interface{}) {}, `{"certificate":"CERT-NEW","certificateChain":"CHAIN-NEW","issuingCaCertificate":"CA","serialNumber":"serial-new","privateKey":"KEY-RENEW","certificateId":"cert-new","certificateRequestId":"req-new"}`, http.StatusOK, log)

	resp, got := runUpdate(t, r, renewalPlan(t, state, state), state)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if got.PrivateKey.ValueString() != "KEY-RENEW" {
		t.Errorf("private_key = %q, want the key from the renew response", got.PrivateKey.ValueString())
	}
	for _, request := range log.all() {
		if strings.HasSuffix(request, "/bundle") {
			t.Errorf("bundle must not be fetched when the renew response has the key, requests = %v", log.all())
		}
	}
}

func TestCertUpdateFieldRenewalWithoutKeyPermissionWarns(t *testing.T) {
	setTestNow(t, "2026-10-25T00:00:00Z")
	state := fieldCertModel()
	state.ManualRenewDaysBeforeExpiry = types.Int64Value(10)
	log := &certRequestLog{}
	r := renewTestServer(t, func(map[string]interface{}) {}, testRenewResponse, http.StatusForbidden, log)

	resp, got := runUpdate(t, r, renewalPlan(t, state, state), state)
	if resp.Diagnostics.HasError() {
		t.Fatalf("a missing private key permission must be a warning, got %v", resp.Diagnostics)
	}
	if resp.Diagnostics.WarningsCount() != 1 || got.Id.ValueString() != "cert-new" || got.PrivateKey.ValueString() != "" {
		t.Errorf("expected the renewed certificate with an empty key and one warning, got %+v %v", got, resp.Diagnostics)
	}
}

func TestCertReadKeepsCertificateWhenLatestRenewalIsGone(t *testing.T) {
	r := newCertTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.Method + " " + req.URL.Path {
		case "GET /api/v1/cert-manager/certificates/cert-old":
			_, _ = fmt.Fprintf(w, `{"certificate":{"id":"cert-old","status":"active","commonName":"api.example.com","serialNumber":"serial-old","notBefore":%q,"notAfter":%q,"renewedByCertificateId":"cert-latest","latestRenewalCertificateId":"cert-latest","orderId":"order-1"}}`, testCertNotBefore, testCertNotAfter)
		case "GET /api/v1/cert-manager/certificates/cert-latest":
			http.Error(w, `{"message":"Certificate not found"}`, http.StatusNotFound)
		default:
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			http.NotFound(w, req)
		}
	})
	resp := runRead(t, r, fieldCertModel())
	if resp.Diagnostics.HasError() {
		t.Fatalf("a vanished renewal must not fail refresh, got %v", resp.Diagnostics)
	}
	got := stateModel(t, resp.State)
	if got.Id.ValueString() != "cert-old" || got.Status.ValueString() != certificateStatusRenewed || got.Certificate.ValueString() != "CERT-OLD" {
		t.Errorf("expected to stay on cert-old marked renewed, got %+v", got)
	}
}

func TestCertRevocationReasonsComeFromOneTable(t *testing.T) {
	if len(SUPPORTED_CERT_REVOCATION_REASONS) != len(certificateRevocationReasons) {
		t.Fatalf("validator list and API map differ: %v vs %v", SUPPORTED_CERT_REVOCATION_REASONS, certificateRevocationReasons)
	}
	for _, name := range SUPPORTED_CERT_REVOCATION_REASONS {
		if certificateRevocationReasons[name] == "" {
			t.Errorf("reason %q has no API value", name)
		}
	}
	if certificateRevocationReasons["aa_compromise"] != "A_A_COMPROMISE" {
		t.Errorf("aa_compromise maps to %q", certificateRevocationReasons["aa_compromise"])
	}
}

func TestCertUpdateWindowRenewalOfExternalCaCSRCertificateSendsTheCSR(t *testing.T) {
	setTestNow(t, "2026-10-25T00:00:00Z")
	state := csrCertModel()
	state.ManualRenewDaysBeforeExpiry = types.Int64Value(10)
	log := &certRequestLog{}
	r := renewTestServerWithCa(t, "external", func(body map[string]interface{}) {
		if body["renewalKeySource"] != "csr" || body["csr"] != testCSR {
			t.Errorf("external CAs reject reuse, so the configured CSR must be sent, got %v", body)
		}
	}, testRenewResponse, http.StatusOK, log)

	resp, got := runUpdate(t, r, renewalPlan(t, state, state), state)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if got.Id.ValueString() != "cert-new" {
		t.Errorf("id = %q, want cert-new", got.Id.ValueString())
	}
}

func TestCertUpdateAsyncRenewalThatTimesOutKeepsTheCertificate(t *testing.T) {
	setTestNow(t, "2026-10-25T00:00:00Z")
	state := fieldCertModel()
	state.ManualRenewDaysBeforeExpiry = types.Int64Value(10)
	state.TimeoutSeconds = types.Int64Value(1)
	r := newCertTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.Method + " " + req.URL.Path {
		case "POST /api/v1/cert-manager/certificates/cert-old/renew":
			_, _ = fmt.Fprint(w, `{"certificate":"","certificateChain":"","issuingCaCertificate":"","serialNumber":"","certificateId":"placeholder","certificateRequestId":"req-slow"}`)
		case "GET /api/v1/cert-manager/certificates/certificate-requests/req-slow":
			_, _ = fmt.Fprint(w, `{"status":"pending"}`)
		default:
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			http.NotFound(w, req)
		}
	})

	resp, got := runUpdate(t, r, renewalPlan(t, state, state), state)
	if resp.Diagnostics.HasError() {
		t.Fatalf("a renewal still pending must not fail the apply, got %v", resp.Diagnostics)
	}
	if resp.Diagnostics.WarningsCount() != 1 || resp.Diagnostics.Warnings()[0].Summary() != "Certificate renewal still pending" {
		t.Errorf("expected one pending warning, got %v", resp.Diagnostics)
	}
	if got.Id.ValueString() != "cert-old" || got.Certificate.ValueString() != "CERT-OLD" || got.ManualRenewDaysBeforeExpiry.ValueInt64() != 10 {
		t.Errorf("the current certificate must stay in state, got %+v", got)
	}
}

type fakePrivateState map[string][]byte

func (f fakePrivateState) GetKey(_ context.Context, key string) ([]byte, diag.Diagnostics) {
	return f[key], nil
}

func TestCertPendingRenewalRequestIdFromPrivateState(t *testing.T) {
	ctx := context.Background()
	if got := pendingRenewalRequestId(ctx, fakePrivateState{}); got != "" {
		t.Errorf("no pending renewal expected, got %q", got)
	}
	if got := pendingRenewalRequestId(ctx, fakePrivateState{certificatePendingRenewalKey: []byte(`"req-slow"`)}); got != "req-slow" {
		t.Errorf("pending renewal = %q, want req-slow", got)
	}
}

func TestCertIssuanceFailureIsNotTreatedAsPending(t *testing.T) {
	var failed, timedOut diag.Diagnostics
	failed.AddError("Certificate issuance failed", "Certificate issuance failed: CA rejected the request")
	timedOut.AddError("Certificate issuance timeout", "did not complete")
	if !certificateIssuanceFailed(failed) || certificateIssuanceFailed(timedOut) {
		t.Error("only an issuance failure may fail the apply; a timeout keeps the renewal pending")
	}
}

func readServerForDeleted(t *testing.T, latest string) *certManagerCertificateResource {
	return newCertTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.Method + " " + req.URL.Path {
		case "GET /api/v1/cert-manager/certificates/cert-old":
			if req.URL.Query().Get("includeDeleted") != "true" {
				t.Errorf("Read must request includeDeleted, got %q", req.URL.RawQuery)
			}
			latestJSON := "null"
			if latest != "" {
				latestJSON = fmt.Sprintf("%q", latest)
			}
			_, _ = fmt.Fprintf(w, `{"certificate":{"id":"cert-old","status":"active","commonName":"api.example.com","serialNumber":"serial-old","notBefore":%q,"notAfter":%q,"deletedAt":"2026-10-10T00:00:00Z","renewedByCertificateId":"cert-latest","latestRenewalCertificateId":%s,"orderId":"order-1"}}`, testCertNotBefore, testCertNotAfter, latestJSON)
		case "GET /api/v1/cert-manager/certificates/cert-latest":
			writeCertDetails(w, "cert-latest", "serial-latest")
		case "GET /api/v1/cert-manager/certificates/cert-latest/bundle":
			_, _ = fmt.Fprint(w, `{"certificate":"CERT-LATEST","certificateChain":"CHAIN-LATEST","privateKey":"KEY-LATEST","serialNumber":"serial-latest"}`)
		default:
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			http.NotFound(w, req)
		}
	})
}

func TestCertReadFollowsRenewalOfADeletedCertificate(t *testing.T) {
	resp := runRead(t, readServerForDeleted(t, "cert-latest"), fieldCertModel())
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if resp.State.Raw.IsNull() {
		t.Fatal("a deleted certificate with a live renewal must follow the renewal, not leave state")
	}
	if got := stateModel(t, resp.State); got.Id.ValueString() != "cert-latest" || got.PrivateKey.ValueString() != "KEY-LATEST" {
		t.Errorf("expected to follow cert-latest, got %+v", got)
	}
}

func TestCertReadRemovesDeletedCertificateWithoutRenewal(t *testing.T) {
	resp := runRead(t, readServerForDeleted(t, ""), fieldCertModel())
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Error("a deleted certificate without a live renewal must be removed from state")
	}
}

func TestCertModifyPlanRejectsWindowNotSmallerThanTTLOnCreate(t *testing.T) {
	model := fieldCertModel()
	model.TTL = types.StringValue("30d")
	model.ManualRenewDaysBeforeExpiry = types.Int64Value(60)
	resp, _ := runModifyPlan(t, &model, nil)
	if !resp.Diagnostics.HasError() {
		t.Fatal("a window of 60 days on a 30 day certificate must fail at plan time on create")
	}
	model.ManualRenewDaysBeforeExpiry = types.Int64Value(10)
	resp, _ = runModifyPlan(t, &model, nil)
	if resp.Diagnostics.HasError() {
		t.Fatalf("a 10 day window on a 30 day certificate is valid, got %v", resp.Diagnostics)
	}
}

func TestCertValidateConfigWarnsOnReasonWithoutDestroyFlag(t *testing.T) {
	ctx := context.Background()
	s := certTestSchema(t)
	for _, tc := range []struct {
		revoke, delete bool
		wantWarning    bool
	}{{false, false, true}, {true, false, false}, {false, true, true}, {true, true, false}} {
		model := configFromModel(fieldCertModel())
		model.RevocationReason = types.StringValue("superseded")
		model.RevokeOnDestroy = types.BoolValue(tc.revoke)
		model.DeleteOnDestroy = types.BoolValue(tc.delete)
		resp := resource.ValidateConfigResponse{}
		(&certManagerCertificateResource{}).ValidateConfig(ctx, resource.ValidateConfigRequest{Config: certConfig(t, s, &model)}, &resp)
		if (resp.Diagnostics.WarningsCount() == 1) != tc.wantWarning {
			t.Errorf("revoke=%t delete=%t: warnings = %v, want warning %t", tc.revoke, tc.delete, resp.Diagnostics, tc.wantWarning)
		}
	}
}
