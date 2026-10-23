package resource

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
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
		{name: "revoke_404_but_certificate_exists_fails", now: "2026-10-05T00:00:00Z", revoke: true, reason: "unspecified", serverStatus: "active", revokeStatus: http.StatusNotFound, wantReason: "UNSPECIFIED", wantRequests: []string{get, revoke, get}, wantError: true},
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
					writeCertDetailsTo(w, "cert-latest", "serial-latest")
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

func TestCertReadRemovesCertificateThatNoLongerExists(t *testing.T) {
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

type fakePrivateState map[string][]byte

func (f fakePrivateState) GetKey(_ context.Context, key string) ([]byte, diag.Diagnostics) {
	return f[key], nil
}

func TestCertPendingRenewalFromPrivateState(t *testing.T) {
	ctx := context.Background()
	if _, ok := readPendingRenewal(ctx, fakePrivateState{}); ok {
		t.Error("no pending renewal expected")
	}
	got, ok := readPendingRenewal(ctx, fakePrivateState{certificatePendingRenewalKey: []byte(`{"requestId":"req-slow","csr":"CSR"}`)})
	if !ok || got.RequestId != "req-slow" || got.CSR != "CSR" {
		t.Errorf("pending renewal = %+v, want req-slow with its CSR", got)
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
			writeCertDetailsTo(w, "cert-latest", "serial-latest")
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

func testKeyMaterial(t *testing.T) (csrPEM, certificatePEM string) {
	commonName := "api.example.com"
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{Subject: pkix.Name{CommonName: commonName}}, key)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: commonName}, NotBefore: time.Now(), NotAfter: time.Now().Add(24 * time.Hour)}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER})), string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}))
}

func TestCertImportedCSRCertificateRenewsWhenANewKeyIsConfigured(t *testing.T) {
	sameCSR, certificate := testKeyMaterial(t)
	otherCSR, _ := testKeyMaterial(t)
	state := fieldCertModel()
	state.CSR = types.StringNull()
	state.Certificate = types.StringValue(certificate)
	state.PrivateKey = types.StringValue("")

	plan := state
	plan.CSR = types.StringValue(sameCSR)
	if certificateCSRChanged(plan, state) {
		t.Error("the CSR the imported certificate was issued from must not trigger a renewal")
	}
	plan.CSR = types.StringValue(otherCSR)
	if !certificateCSRChanged(plan, state) {
		t.Error("a CSR with a different key must renew the imported certificate")
	}
}

func TestCertModifyPlanReplacementSkipsRenewalChecks(t *testing.T) {
	setTestNow(t, "2026-10-05T00:00:00Z")
	state := csrCertModel()
	state.Status = types.StringValue("revoked")
	plan := state
	plan.CSR = types.StringValue(testRotatedCSR)
	plan.ProfileId = types.StringValue("profile-2")

	resp, req := runModifyPlan(t, &plan, &state)
	if resp.Diagnostics.HasError() {
		t.Fatalf("a planned replacement must not run the renewal checks, got %v", resp.Diagnostics)
	}
	if !resp.Plan.Raw.Equal(req.Plan.Raw) {
		t.Error("a planned replacement must leave the plan to the framework")
	}
}

func TestCertModifyPlanResumesPendingRenewalWithTheSameCSR(t *testing.T) {
	setTestNow(t, "2026-10-05T00:00:00Z")
	s := certTestSchema(t)
	state := csrCertModel()
	plan := state
	plan.CSR = types.StringValue(testRotatedCSR)
	config := configFromModel(plan)
	run := func(pendingCSR string) resource.ModifyPlanResponse {
		req := resource.ModifyPlanRequest{Plan: certPlan(t, s, &plan), State: certState(t, s, &state), Config: certConfig(t, s, &config)}
		resp := resource.ModifyPlanResponse{Plan: req.Plan}
		value, _ := json.Marshal(pendingRenewal{RequestId: "req-slow", CSR: pendingCSR})
		newCertTestResource(t, failOnRequest(t)).modifyPlan(context.Background(), req, fakePrivateState{certificatePendingRenewalKey: value}, &resp)
		return resp
	}
	resp := run(testRotatedCSR)
	if resp.Diagnostics.HasError() {
		t.Fatalf("the pending renewal of the same CSR must be resumed, got %v", resp.Diagnostics)
	}
	assertRenewalPlanned(t, resp)
	if resp = run(testCSR + "x"); !resp.Diagnostics.HasError() {
		t.Error("a different CSR while another renewal is pending must fail the plan")
	}
}

func TestCertUpdateFailedRenewalKeepsThePriorState(t *testing.T) {
	setTestNow(t, "2026-10-25T00:00:00Z")
	state := fieldCertModel()
	state.ManualRenewDaysBeforeExpiry = types.Int64Value(10)
	r := newCertTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		http.Error(w, `{"message":"CA rejected the renewal"}`, http.StatusBadRequest)
	})
	resp, got := runUpdate(t, r, renewalPlan(t, state, state), state)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected the renewal error")
	}
	if !resp.State.Raw.IsFullyKnown() || got.Id.ValueString() != "cert-old" || got.Certificate.ValueString() != "CERT-OLD" {
		t.Errorf("a failed renewal must keep the prior certificate in state, got %+v", got)
	}
}

func TestCertReadKeepsDeletedButUnrevokedCertificateWhenRevokeOnDestroyIsSet(t *testing.T) {
	r := newCertTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"certificate":{"id":"cert-old","status":"active","caId":"ca-1","commonName":"api.example.com","notBefore":%q,"notAfter":%q,"deletedAt":"2026-10-10T00:00:00Z","latestRenewalCertificateId":null}}`, testCertNotBefore, testCertNotAfter)
	})
	setTestNow(t, "2026-10-12T00:00:00Z")
	state := fieldCertModel()
	state.RevokeOnDestroy = types.BoolValue(true)
	resp := runRead(t, r, state)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if resp.State.Raw.IsNull() {
		t.Fatal("a deleted but unrevoked certificate must stay in state so destroy can revoke it")
	}
	got := stateModel(t, resp.State)
	if got.Status.ValueString() != certificateStatusDeleted {
		t.Errorf("status = %q, want deleted", got.Status.ValueString())
	}

	planResp, _ := runModifyPlan(t, &got, &got)
	if len(planResp.RequiresReplace) != 1 || !planResp.RequiresReplace[0].Equal(path.Root("status")) {
		t.Errorf("a deleted certificate must be replaced, RequiresReplace = %v", planResp.RequiresReplace)
	}

	state.RevokeOnDestroy = types.BoolValue(false)
	if resp = runRead(t, r, state); !resp.State.Raw.IsNull() {
		t.Error("without revoke_on_destroy a deleted certificate leaves state as before")
	}
}

func TestCertDeleteSeesARenewalThatFinishesWhileCancelling(t *testing.T) {
	setTestNow(t, "2026-10-05T00:00:00Z")
	cancelled := false
	log := &certRequestLog{}
	r := newCertTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		log.add(req.Method + " " + req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch req.Method + " " + req.URL.Path {
		case "POST /api/v1/cert-manager/certificates/certificate-requests/req-slow/cancel":
			cancelled = true
			_, _ = fmt.Fprint(w, `{"status":"issued","cancelled":false,"errorMessage":null}`)
		case "GET /api/v1/cert-manager/certificates/cert-old":
			renewedBy := ""
			if cancelled {
				renewedBy = "cert-next"
			}
			_, _ = fmt.Fprintf(w, `{"certificate":{"id":"cert-old","status":"active","caId":"ca-1","notAfter":"2026-10-31T00:00:00Z","renewedByCertificateId":%q}}`, renewedBy)
		case "GET /api/v1/cert-manager/certificates/cert-next":
			_, _ = fmt.Fprint(w, `{"certificate":{"id":"cert-next","status":"active","caId":"ca-1","notAfter":"2026-11-30T00:00:00Z","renewedFromCertificateId":"cert-old"}}`)
		case "POST /api/v1/cert-manager/certificates/cert-old/revoke", "POST /api/v1/cert-manager/certificates/cert-next/revoke":
			_, _ = fmt.Fprint(w, `{"message":"ok"}`)
		default:
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			http.NotFound(w, req)
		}
	})
	state := fieldCertModel()
	state.RevokeOnDestroy = types.BoolValue(true)
	s := certTestSchema(t)
	priorState := certState(t, s, &state)
	resp := resource.DeleteResponse{State: priorState}
	value, _ := json.Marshal(pendingRenewal{RequestId: "req-slow"})
	r.delete(context.Background(), resource.DeleteRequest{State: priorState}, fakePrivateState{certificatePendingRenewalKey: value}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if !strings.Contains(strings.Join(log.all(), ","), "POST /api/v1/cert-manager/certificates/cert-next/revoke") {
		t.Errorf("a renewal issued while cancelling must be revoked too, requests = %v", log.all())
	}
}

func renewalChainServer(t *testing.T, length int, loop bool, log *certRequestLog) *certManagerCertificateResource {
	return newCertTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		id := strings.TrimPrefix(req.URL.Path, "/api/v1/cert-manager/certificates/")
		switch {
		case req.Method == "GET":
			index := 0
			if id != "cert-old" {
				_, _ = fmt.Sscanf(id, "cert-%d", &index)
			}
			previous := ""
			if index < length {
				previous = fmt.Sprintf("cert-%d", index+1)
			}
			if loop && index == length {
				previous = "cert-old"
			}
			_, _ = fmt.Fprintf(w, `{"certificate":{"id":%q,"status":"active","caId":"ca-1","notAfter":"2026-10-31T00:00:00Z","renewedFromCertificateId":%q}}`, id, previous)
		case req.Method == "POST" && strings.HasSuffix(id, "/revoke"):
			log.add(id)
			_, _ = fmt.Fprint(w, `{"message":"ok"}`)
		default:
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			http.NotFound(w, req)
		}
	})
}

func TestCertDeleteWalksTheWholeRenewalHistory(t *testing.T) {
	setTestNow(t, "2026-10-05T00:00:00Z")
	log := &certRequestLog{}
	state := fieldCertModel()
	state.RevokeOnDestroy = types.BoolValue(true)
	resp := runDelete(t, renewalChainServer(t, 150, false, log), state)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if got := len(log.all()); got != 151 {
		t.Errorf("revoked %d certificates, want all 151 generations", got)
	}
}

func TestCertDeleteFailsOnALoopingRenewalHistory(t *testing.T) {
	setTestNow(t, "2026-10-05T00:00:00Z")
	log := &certRequestLog{}
	state := fieldCertModel()
	state.RevokeOnDestroy = types.BoolValue(true)
	resp := runDelete(t, renewalChainServer(t, 3, true, log), state)
	if !resp.Diagnostics.HasError() || len(log.all()) != 0 {
		t.Errorf("a looping history must fail before revoking anything, got %v and revokes %v", resp.Diagnostics, log.all())
	}
}

func TestCertDeleteRevokeAndDeleteRaces(t *testing.T) {
	setTestNow(t, "2026-10-05T00:00:00Z")
	gone := false
	r := newCertTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.Method + " " + req.URL.Path {
		case "GET /api/v1/cert-manager/certificates/cert-old":
			if gone {
				http.Error(w, `{"message":"Certificate not found"}`, http.StatusNotFound)
				return
			}
			_, _ = fmt.Fprint(w, `{"certificate":{"id":"cert-old","status":"active","caId":"ca-1","notAfter":"2026-10-31T00:00:00Z","renewedFromCertificateId":"cert-prev"}}`)
		case "GET /api/v1/cert-manager/certificates/cert-prev":
			_, _ = fmt.Fprint(w, `{"certificate":{"id":"cert-prev","status":"active","caId":"ca-1","notAfter":"2026-10-20T00:00:00Z","renewedByCertificateId":"cert-old"}}`)
		case "POST /api/v1/cert-manager/certificates/cert-prev/revoke":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprint(w, `{"message":"Certificate 'cert-prev' is already revoked."}`)
		case "POST /api/v1/cert-manager/certificates/cert-old/revoke":
			gone = true
			http.Error(w, `{"message":"Certificate not found"}`, http.StatusNotFound)
		case "DELETE /api/v1/cert-manager/certificates/cert-prev":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprint(w, `{"message":"Certificate 'cert-prev' was already deleted on 2026-10-05T00:00:00.000Z."}`)
		case "DELETE /api/v1/cert-manager/certificates/cert-old":
			http.Error(w, `{"message":"Certificate not found"}`, http.StatusNotFound)
		default:
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			http.NotFound(w, req)
		}
	})
	state := fieldCertModel()
	state.RevokeOnDestroy = types.BoolValue(true)
	state.DeleteOnDestroy = types.BoolValue(true)
	resp := runDelete(t, r, state)
	if resp.Diagnostics.HasError() {
		t.Errorf("already revoked, already deleted and really gone must all count as done, got %v", resp.Diagnostics)
	}
}

func TestCertDeleteReportsTheReadErrorAfterARevoke404(t *testing.T) {
	setTestNow(t, "2026-10-05T00:00:00Z")
	reads := 0
	r := newCertTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.Method + " " + req.URL.Path {
		case "GET /api/v1/cert-manager/certificates/cert-old":
			reads++
			if reads > 1 {
				http.Error(w, `{"message":"Something went wrong"}`, http.StatusInternalServerError)
				return
			}
			_, _ = fmt.Fprint(w, `{"certificate":{"id":"cert-old","status":"active","caId":"ca-1","notAfter":"2026-10-31T00:00:00Z"}}`)
		case "POST /api/v1/cert-manager/certificates/cert-old/revoke":
			http.Error(w, `{"message":"App connection not found"}`, http.StatusNotFound)
		default:
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			http.NotFound(w, req)
		}
	})
	state := fieldCertModel()
	state.RevokeOnDestroy = types.BoolValue(true)
	resp := runDelete(t, r, state)
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "reading it again failed") {
		t.Errorf("the read error must be reported, got %v", resp.Diagnostics)
	}
}

const certAPI = "/api/v1/cert-manager/certificates/"

type apiReply struct {
	status int
	body   string
	check  func(body map[string]interface{})
}

func reply(body string) apiReply {
	return apiReply{body: body}
}

func fakeAPI(t *testing.T, log *certRequestLog, routes map[string]apiReply) *certManagerCertificateResource {
	t.Helper()
	return newCertTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		key := req.Method + " " + req.URL.Path
		if log != nil {
			log.add(key)
		}
		route, ok := routes[key]
		if !ok {
			t.Errorf("unexpected request: %s", key)
			http.NotFound(w, req)
			return
		}
		if route.check != nil {
			var body map[string]interface{}
			_ = json.NewDecoder(req.Body).Decode(&body)
			route.check(body)
		}
		w.Header().Set("Content-Type", "application/json")
		if route.status != 0 {
			w.WriteHeader(route.status)
		}
		_, _ = fmt.Fprint(w, route.body)
	})
}

func certDetailsJSON(id, serial string) string {
	return fmt.Sprintf(`{"certificate":{"id":%q,"status":"active","commonName":"api.example.com","serialNumber":%q,"notBefore":"2026-10-25T00:00:00Z","notAfter":"2026-11-24T00:00:00Z","keyAlgorithm":"RSA_2048","signatureAlgorithm":"RSA-SHA256","profileId":"profile-1","applicationId":"app-1","orderId":"order-1"}}`, id, serial)
}

func writeCertDetailsTo(w http.ResponseWriter, id, serial string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprint(w, certDetailsJSON(id, serial))
}

func caLookup(caType string) apiReply {
	return reply(fmt.Sprintf(`{"certificate":{"id":"cert-old","status":"active","caId":"ca-1","caType":%q}}`, caType))
}

const (
	syncRenewResponse  = `{"certificate":"CERT-NEW","certificateChain":"CHAIN-NEW","issuingCaCertificate":"CA","serialNumber":"serial-new","certificateId":"cert-new","certificateRequestId":"req-new"}`
	asyncRenewResponse = `{"certificate":"","certificateChain":"","issuingCaCertificate":"","serialNumber":"","certificateId":"placeholder","certificateRequestId":"req-slow"}`
	newBundle          = `{"certificate":"CERT-NEW","certificateChain":"CHAIN-NEW","privateKey":"KEY-BUNDLE","serialNumber":"serial-new"}`
)

func TestCertUpdateRenewal(t *testing.T) {
	noKeySourceOrCSR := func(t *testing.T) func(map[string]interface{}) {
		return func(body map[string]interface{}) {
			if _, ok := body["renewalKeySource"]; ok {
				t.Errorf("field certificates must not send a key source, got %v", body)
			}
			if _, ok := body["csr"]; ok {
				t.Errorf("field certificates must not send a CSR, got %v", body)
			}
		}
	}
	tests := []struct {
		name         string
		csrCert      bool
		rotateCSR    bool
		renewBody    string
		renewCheck   func(t *testing.T) func(map[string]interface{})
		routes       map[string]apiReply
		wantId       string
		wantKey      string
		wantWarnings int
		wantRequests []string
	}{
		{
			name: "csr_rotated_sends_the_new_normalized_csr", csrCert: true, rotateCSR: true, renewBody: syncRenewResponse,
			renewCheck: func(t *testing.T) func(map[string]interface{}) {
				return func(body map[string]interface{}) {
					if body["renewalKeySource"] != "csr" || body["csr"] != testRotatedCSR {
						t.Errorf("want key source csr with the normalized new CSR, got %v", body)
					}
					if _, ok := body["attributes"]; ok {
						t.Errorf("renewals copy the original certificate and must not send attributes, got %v", body)
					}
				}
			},
			routes:       map[string]apiReply{"GET " + certAPI + "cert-new": reply(certDetailsJSON("cert-new", "serial-new"))},
			wantId:       "cert-new",
			wantRequests: []string{"POST " + certAPI + "cert-old/renew", "GET " + certAPI + "cert-new"},
		},
		{
			name: "csr_window_on_internal_ca_reuses_the_key", csrCert: true, renewBody: syncRenewResponse,
			renewCheck: func(t *testing.T) func(map[string]interface{}) {
				return func(body map[string]interface{}) {
					if body["renewalKeySource"] != "reuse" {
						t.Errorf("want reuse so the renewal is not a certificate edit, got %v", body)
					}
					if _, ok := body["csr"]; ok {
						t.Errorf("an unchanged CSR must not be sent, got %v", body)
					}
					if _, ok := body["attributes"]; ok {
						t.Errorf("attributes must not be sent, got %v", body)
					}
				}
			},
			routes:       map[string]apiReply{"GET " + certAPI + "cert-old": caLookup("internal"), "GET " + certAPI + "cert-new": reply(certDetailsJSON("cert-new", "serial-new"))},
			wantId:       "cert-new",
			wantRequests: []string{"GET " + certAPI + "cert-old", "POST " + certAPI + "cert-old/renew", "GET " + certAPI + "cert-new"},
		},
		{
			name: "csr_window_on_external_ca_sends_the_configured_csr", csrCert: true, renewBody: syncRenewResponse,
			renewCheck: func(t *testing.T) func(map[string]interface{}) {
				return func(body map[string]interface{}) {
					if body["renewalKeySource"] != "csr" || body["csr"] != testCSR {
						t.Errorf("external CAs reject reuse, so the configured CSR must be sent, got %v", body)
					}
				}
			},
			routes:       map[string]apiReply{"GET " + certAPI + "cert-old": caLookup("external"), "GET " + certAPI + "cert-new": reply(certDetailsJSON("cert-new", "serial-new"))},
			wantId:       "cert-new",
			wantRequests: []string{"GET " + certAPI + "cert-old", "POST " + certAPI + "cert-old/renew", "GET " + certAPI + "cert-new"},
		},
		{
			name: "field_window_fetches_the_new_key", renewBody: syncRenewResponse, renewCheck: noKeySourceOrCSR,
			routes:       map[string]apiReply{"GET " + certAPI + "cert-new": reply(certDetailsJSON("cert-new", "serial-new")), "GET " + certAPI + "cert-new/bundle": reply(newBundle)},
			wantId:       "cert-new",
			wantKey:      "KEY-BUNDLE",
			wantRequests: []string{"POST " + certAPI + "cert-old/renew", "GET " + certAPI + "cert-new", "GET " + certAPI + "cert-new/bundle"},
		},
		{
			name:         "field_window_uses_the_key_from_the_renew_response",
			renewBody:    `{"certificate":"CERT-NEW","certificateChain":"CHAIN-NEW","issuingCaCertificate":"CA","serialNumber":"serial-new","privateKey":"KEY-RENEW","certificateId":"cert-new","certificateRequestId":"req-new"}`,
			renewCheck:   noKeySourceOrCSR,
			routes:       map[string]apiReply{"GET " + certAPI + "cert-new": reply(certDetailsJSON("cert-new", "serial-new"))},
			wantId:       "cert-new",
			wantKey:      "KEY-RENEW",
			wantRequests: []string{"POST " + certAPI + "cert-old/renew", "GET " + certAPI + "cert-new"},
		},
		{
			name: "field_window_without_key_permission_warns", renewBody: syncRenewResponse, renewCheck: noKeySourceOrCSR,
			routes: map[string]apiReply{
				"GET " + certAPI + "cert-new":        reply(certDetailsJSON("cert-new", "serial-new")),
				"GET " + certAPI + "cert-new/bundle": {status: http.StatusForbidden, body: `{"message":"forbidden"}`},
			},
			wantId:       "cert-new",
			wantWarnings: 1,
			wantRequests: []string{"POST " + certAPI + "cert-old/renew", "GET " + certAPI + "cert-new", "GET " + certAPI + "cert-new/bundle"},
		},
		{
			name: "async_renewal_ignores_the_placeholder_id", renewBody: asyncRenewResponse, renewCheck: noKeySourceOrCSR,
			routes: map[string]apiReply{
				"GET " + certAPI + "certificate-requests/req-slow": reply(`{"status":"issued","certificateId":"cert-real","certificate":"CERT-NEW","certificateChain":"CHAIN-NEW","serialNumber":"serial-new"}`),
				"GET " + certAPI + "cert-real":                     reply(certDetailsJSON("cert-real", "serial-new")),
				"GET " + certAPI + "cert-real/bundle":              reply(newBundle),
			},
			wantId:       "cert-real",
			wantKey:      "KEY-BUNDLE",
			wantRequests: []string{"POST " + certAPI + "cert-old/renew", "GET " + certAPI + "certificate-requests/req-slow", "GET " + certAPI + "cert-real", "GET " + certAPI + "cert-real/bundle"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := fieldCertModel()
			if tt.csrCert {
				state = csrCertModel()
			}
			plan := state
			if tt.rotateCSR {
				setTestNow(t, "2026-10-05T00:00:00Z")
				plan.CSR = types.StringValue(strings.ReplaceAll(testRotatedCSR, "\n", "\r\n") + "\n")
			} else {
				setTestNow(t, "2026-10-25T00:00:00Z")
				state.ManualRenewDaysBeforeExpiry = types.Int64Value(10)
				plan = state
			}
			routes := map[string]apiReply{"POST " + certAPI + "cert-old/renew": {body: tt.renewBody, check: tt.renewCheck(t)}}
			for k, v := range tt.routes {
				routes[k] = v
			}
			log := &certRequestLog{}
			resp, got := runUpdate(t, fakeAPI(t, log, routes), renewalPlan(t, plan, state), state)
			if resp.Diagnostics.HasError() || resp.Diagnostics.WarningsCount() != tt.wantWarnings {
				t.Fatalf("diagnostics = %v, want no error and %d warnings", resp.Diagnostics, tt.wantWarnings)
			}
			if got.Id.ValueString() != tt.wantId || got.PrivateKey.ValueString() != tt.wantKey || got.Certificate.ValueString() != "CERT-NEW" ||
				got.SerialNumber.ValueString() != "serial-new" || got.Status.ValueString() != "active" || got.NotAfter.ValueString() != "2026-11-24T00:00:00Z" {
				t.Errorf("unexpected state after renewal: %+v", got)
			}
			if strings.Join(log.all(), ",") != strings.Join(tt.wantRequests, ",") {
				t.Errorf("requests = %v, want %v", log.all(), tt.wantRequests)
			}
		})
	}
}

func TestCertUpdateRenewalStillPendingAtTimeout(t *testing.T) {
	for _, tt := range []struct {
		name      string
		rotateCSR bool
		wantError bool
	}{
		{name: "window_renewal_warns_and_keeps_the_certificate"},
		{name: "csr_renewal_errors_and_keeps_the_old_csr", rotateCSR: true, wantError: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			setTestNow(t, "2026-10-25T00:00:00Z")
			state := fieldCertModel()
			state.ManualRenewDaysBeforeExpiry = types.Int64Value(10)
			if tt.rotateCSR {
				setTestNow(t, "2026-10-05T00:00:00Z")
				state = csrCertModel()
			}
			state.TimeoutSeconds = types.Int64Value(1)
			plan := state
			if tt.rotateCSR {
				plan.CSR = types.StringValue(testRotatedCSR)
			}
			r := fakeAPI(t, nil, map[string]apiReply{
				"POST " + certAPI + "cert-old/renew":               reply(asyncRenewResponse),
				"GET " + certAPI + "certificate-requests/req-slow": reply(`{"status":"pending"}`),
			})
			resp, got := runUpdate(t, r, renewalPlan(t, plan, state), state)
			if resp.Diagnostics.HasError() != tt.wantError {
				t.Fatalf("error = %v, want error %t", resp.Diagnostics, tt.wantError)
			}
			summaries := append(resp.Diagnostics.Errors(), resp.Diagnostics.Warnings()...)
			if len(summaries) != 1 || summaries[0].Summary() != "Certificate renewal still pending" {
				t.Errorf("want one pending diagnostic, got %v", resp.Diagnostics)
			}
			if got.Id.ValueString() != "cert-old" || got.Certificate.ValueString() != "CERT-OLD" || got.CSR.ValueString() != state.CSR.ValueString() {
				t.Errorf("the current certificate and the old CSR must stay in state, got %+v", got)
			}
		})
	}
}

func TestCertModifyPlanRenewalWindowSize(t *testing.T) {
	setTestNow(t, "2026-10-02T00:00:00Z")
	for _, tc := range []struct {
		name      string
		create    bool
		stateTTL  types.String
		planTTL   types.String
		days      int64
		wantError bool
	}{
		{name: "equal_to_lifetime", stateTTL: types.StringValue("30d"), planTTL: types.StringValue("30d"), days: 30, wantError: true},
		{name: "real_lifetime_wins_over_unchanged_ttl", stateTTL: types.StringValue("2m"), planTTL: types.StringValue("2m"), days: 31, wantError: true},
		{name: "no_ttl_uses_real_lifetime", stateTTL: types.StringNull(), planTTL: types.StringNull(), days: 45, wantError: true},
		{name: "new_ttl_in_weeks", stateTTL: types.StringValue("30d"), planTTL: types.StringValue("4w"), days: 28, wantError: true},
		{name: "new_ttl_in_minutes", stateTTL: types.StringValue("30d"), planTTL: types.StringValue("90m"), days: 1, wantError: true},
		{name: "new_ttl_in_hours", stateTTL: types.StringValue("30d"), planTTL: types.StringValue("12h"), days: 1, wantError: true},
		{name: "smaller_than_real_lifetime", stateTTL: types.StringValue("720h"), planTTL: types.StringValue("720h"), days: 29},
		{name: "create_larger_than_ttl", create: true, planTTL: types.StringValue("30d"), days: 60, wantError: true},
		{name: "create_smaller_than_ttl", create: true, planTTL: types.StringValue("30d"), days: 10},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := fieldCertModel()
			state.TTL = tc.stateTTL
			plan := state
			plan.TTL = tc.planTTL
			plan.ManualRenewDaysBeforeExpiry = types.Int64Value(tc.days)
			var resp resource.ModifyPlanResponse
			if tc.create {
				resp, _ = runModifyPlan(t, &plan, nil)
			} else {
				resp, _ = runModifyPlan(t, &plan, &state)
			}
			if resp.Diagnostics.HasError() != tc.wantError {
				t.Errorf("error = %v, want error %t", resp.Diagnostics, tc.wantError)
			}
		})
	}
}

func TestCertReadCertificateMaterial(t *testing.T) {
	for _, tt := range []struct {
		name         string
		model        func() certManagerCertificateResourceModel
		hasKey       bool
		private      fakePrivateState
		routes       map[string]apiReply
		wantKey      string
		wantRequests []string
	}{
		{
			name: "import_reads_the_bundle",
			model: func() certManagerCertificateResourceModel {
				m := fieldCertModel()
				m.Certificate, m.CertificateChain, m.PrivateKey = types.StringNull(), types.StringNull(), types.StringNull()
				return m
			},
			hasKey:       true,
			routes:       map[string]apiReply{"GET " + certAPI + "cert-old/bundle": reply(`{"certificate":"CERT-OLD","certificateChain":"CHAIN-OLD","privateKey":"KEY-OLD","serialNumber":"serial-old"}`)},
			wantKey:      "KEY-OLD",
			wantRequests: []string{"GET " + certAPI + "cert-old", "GET " + certAPI + "cert-old/bundle"},
		},
		{
			name: "no_stored_key_reads_the_body_without_a_warning",
			model: func() certManagerCertificateResourceModel {
				m := csrCertModel()
				m.CSR, m.Certificate = types.StringNull(), types.StringNull()
				return m
			},
			routes:       map[string]apiReply{"GET " + certAPI + "cert-old/certificate": reply(`{"certificate":"CERT-OLD","certificateChain":"CHAIN-OLD","serialNumber":"serial-old"}`)},
			wantRequests: []string{"GET " + certAPI + "cert-old", "GET " + certAPI + "cert-old/certificate"},
		},
		{
			name: "missing_key_flag_reads_the_key_again",
			model: func() certManagerCertificateResourceModel {
				m := fieldCertModel()
				m.PrivateKey = types.StringValue("")
				return m
			},
			hasKey:       true,
			private:      fakePrivateState{certificateMissingKeyKey: []byte("true")},
			routes:       map[string]apiReply{"GET " + certAPI + "cert-old/bundle": reply(`{"certificate":"CERT-OLD","certificateChain":"CHAIN-OLD","privateKey":"KEY-RECOVERED","serialNumber":"serial-old"}`)},
			wantKey:      "KEY-RECOVERED",
			wantRequests: []string{"GET " + certAPI + "cert-old", "GET " + certAPI + "cert-old/bundle"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			routes := map[string]apiReply{"GET " + certAPI + "cert-old": reply(fmt.Sprintf(`{"certificate":{"id":"cert-old","status":"active","commonName":"api.example.com","notBefore":%q,"notAfter":%q,"hasPrivateKey":%t}}`, testCertNotBefore, testCertNotAfter, tt.hasKey))}
			for k, v := range tt.routes {
				routes[k] = v
			}
			log := &certRequestLog{}
			s := certTestSchema(t)
			state := tt.model()
			priorState := certState(t, s, &state)
			resp := resource.ReadResponse{State: priorState}
			private := tt.private
			if private == nil {
				private = fakePrivateState{}
			}
			fakeAPI(t, log, routes).read(context.Background(), resource.ReadRequest{State: priorState}, private, &resp)
			if resp.Diagnostics.HasError() || resp.Diagnostics.WarningsCount() != 0 {
				t.Fatalf("diagnostics = %v, want none", resp.Diagnostics)
			}
			got := stateModel(t, resp.State)
			if got.Certificate.ValueString() != "CERT-OLD" || got.CertificateChain.ValueString() != "CHAIN-OLD" || got.PrivateKey.ValueString() != tt.wantKey {
				t.Errorf("unexpected material: %+v", got)
			}
			if strings.Join(log.all(), ",") != strings.Join(tt.wantRequests, ",") {
				t.Errorf("requests = %v, want %v", log.all(), tt.wantRequests)
			}
		})
	}
}

func TestCertReadRenewalThatIsGone(t *testing.T) {
	for _, tt := range []struct {
		name   string
		latest string
		routes map[string]apiReply
	}{
		{name: "renewal_no_longer_reported", latest: "null"},
		{name: "reported_renewal_returns_404", latest: `"cert-latest"`, routes: map[string]apiReply{"GET " + certAPI + "cert-latest": {status: http.StatusNotFound, body: `{"message":"Certificate not found"}`}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			routes := map[string]apiReply{"GET " + certAPI + "cert-old": reply(fmt.Sprintf(`{"certificate":{"id":"cert-old","status":"active","commonName":"api.example.com","serialNumber":"serial-old","notBefore":%q,"notAfter":%q,"renewedByCertificateId":"cert-gone","latestRenewalCertificateId":%s,"orderId":"order-1"}}`, testCertNotBefore, testCertNotAfter, tt.latest))}
			for k, v := range tt.routes {
				routes[k] = v
			}
			resp := runRead(t, fakeAPI(t, nil, routes), fieldCertModel())
			if resp.Diagnostics.HasError() {
				t.Fatalf("a vanished renewal must not fail refresh, got %v", resp.Diagnostics)
			}
			got := stateModel(t, resp.State)
			if got.Id.ValueString() != "cert-old" || got.Status.ValueString() != certificateStatusRenewed || got.Certificate.ValueString() != "CERT-OLD" {
				t.Errorf("expected to stay on cert-old marked renewed, got %+v", got)
			}
		})
	}
}

func TestCertReadClearsSubjectFieldsTheFollowedRenewalDropped(t *testing.T) {
	r := fakeAPI(t, nil, map[string]apiReply{
		"GET " + certAPI + "cert-old":           reply(fmt.Sprintf(`{"certificate":{"id":"cert-old","status":"active","commonName":"api.example.com","notBefore":%q,"notAfter":%q,"subjectOrganization":"Example Corp","renewedByCertificateId":"cert-latest","latestRenewalCertificateId":"cert-latest"}}`, testCertNotBefore, testCertNotAfter)),
		"GET " + certAPI + "cert-latest":        reply(certDetailsJSON("cert-latest", "serial-latest")),
		"GET " + certAPI + "cert-latest/bundle": reply(`{"certificate":"CERT-LATEST","certificateChain":"CHAIN-LATEST","privateKey":"KEY-LATEST","serialNumber":"serial-latest"}`),
	})
	state := fieldCertModel()
	state.Organization = types.StringValue("Example Corp")
	resp := runRead(t, r, state)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if got := stateModel(t, resp.State); got.Id.ValueString() != "cert-latest" || !got.Organization.IsNull() {
		t.Errorf("following a renewal without an organization must clear it, got id %v organization %v", got.Id, got.Organization)
	}
}

func TestCertReadCopiesThePendingCSROnlyWhenTheRenewalUsesItsKey(t *testing.T) {
	pendingCSR, matchingCertificate := testKeyMaterial(t)
	_, otherCertificate := testKeyMaterial(t)
	for _, tt := range []struct {
		name        string
		certificate string
		wantCSR     string
	}{
		{name: "renewal_from_the_pending_csr", certificate: matchingCertificate, wantCSR: pendingCSR},
		{name: "renewal_from_another_key", certificate: otherCertificate, wantCSR: testCSR},
	} {
		t.Run(tt.name, func(t *testing.T) {
			bundle, _ := json.Marshal(map[string]string{"certificate": tt.certificate, "certificateChain": "CHAIN", "privateKey": "", "serialNumber": "serial-latest"})
			r := fakeAPI(t, nil, map[string]apiReply{
				"GET " + certAPI + "cert-old":           reply(fmt.Sprintf(`{"certificate":{"id":"cert-old","status":"active","commonName":"api.example.com","notBefore":%q,"notAfter":%q,"renewedByCertificateId":"cert-latest","latestRenewalCertificateId":"cert-latest"}}`, testCertNotBefore, testCertNotAfter)),
				"GET " + certAPI + "cert-latest":        reply(certDetailsJSON("cert-latest", "serial-latest")),
				"GET " + certAPI + "cert-latest/bundle": reply(string(bundle)),
			})
			s := certTestSchema(t)
			state := csrCertModel()
			priorState := certState(t, s, &state)
			resp := resource.ReadResponse{State: priorState}
			pending, _ := json.Marshal(pendingRenewal{RequestId: "req-slow", CSR: pendingCSR})
			r.read(context.Background(), resource.ReadRequest{State: priorState}, fakePrivateState{certificatePendingRenewalKey: pending}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			if got := stateModel(t, resp.State); got.Id.ValueString() != "cert-latest" || got.CSR.ValueString() != tt.wantCSR {
				t.Errorf("csr = %q, want %q", got.CSR.ValueString(), tt.wantCSR)
			}
		})
	}
}

func TestCertImportedCSRCertificateWithoutCSRInConfigRenewsWithReuse(t *testing.T) {
	setTestNow(t, "2026-10-25T00:00:00Z")
	ctx := context.Background()
	s := certTestSchema(t)
	state := csrCertModel()
	state.CSR = types.StringNull()
	state.ManualRenewDaysBeforeExpiry = types.Int64Value(10)
	plan := state
	config := configFromModel(plan)
	private := fakePrivateState{certificateIssuedFromCSRKey: []byte("true")}

	planReq := resource.ModifyPlanRequest{Plan: certPlan(t, s, &plan), State: certState(t, s, &state), Config: certConfig(t, s, &config)}
	planResp := resource.ModifyPlanResponse{Plan: planReq.Plan}
	newCertTestResource(t, failOnRequest(t)).modifyPlan(ctx, planReq, private, &planResp)
	assertRenewalPlanned(t, planResp)

	r := fakeAPI(t, nil, map[string]apiReply{
		"GET " + certAPI + "cert-old": caLookup("internal"),
		"POST " + certAPI + "cert-old/renew": {body: syncRenewResponse, check: func(body map[string]interface{}) {
			if body["renewalKeySource"] != "reuse" {
				t.Errorf("an imported CSR certificate must renew with reuse, got %v", body)
			}
		}},
		"GET " + certAPI + "cert-new": reply(certDetailsJSON("cert-new", "serial-new")),
	})
	priorState := certState(t, s, &state)
	updateResp := resource.UpdateResponse{State: priorState}
	r.update(ctx, resource.UpdateRequest{Plan: planResp.Plan, State: priorState}, private, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatal(updateResp.Diagnostics)
	}
	if got := stateModel(t, updateResp.State); got.Id.ValueString() != "cert-new" {
		t.Errorf("id = %q, want cert-new", got.Id.ValueString())
	}
}

func TestCertUpdateResumesPendingRenewalInsteadOfRenewingAgain(t *testing.T) {
	setTestNow(t, "2026-10-05T00:00:00Z")
	state := csrCertModel()
	plan := state
	plan.CSR = types.StringValue(testRotatedCSR)
	log := &certRequestLog{}
	r := fakeAPI(t, log, map[string]apiReply{
		"GET " + certAPI + "certificate-requests/req-slow": reply(`{"status":"issued","certificateId":"cert-new","certificate":"CERT-NEW","certificateChain":"CHAIN-NEW","serialNumber":"serial-new"}`),
		"GET " + certAPI + "cert-new":                      reply(certDetailsJSON("cert-new", "serial-new")),
	})
	s := certTestSchema(t)
	priorState := certState(t, s, &state)
	resp := resource.UpdateResponse{State: priorState}
	value, _ := json.Marshal(pendingRenewal{RequestId: "req-slow", CSR: testRotatedCSR})
	r.update(context.Background(), resource.UpdateRequest{Plan: renewalPlan(t, plan, state), State: priorState}, fakePrivateState{certificatePendingRenewalKey: value}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if strings.Contains(strings.Join(log.all(), ","), "/renew") {
		t.Fatalf("a pending renewal must be resumed, not requested again: %v", log.all())
	}
	if got := stateModel(t, resp.State); got.Id.ValueString() != "cert-new" {
		t.Errorf("id = %q, want cert-new from the resumed renewal", got.Id.ValueString())
	}
}

func TestCertRejectedRequestCountsAsFailedIssuance(t *testing.T) {
	r := fakeAPI(t, nil, map[string]apiReply{"GET " + certAPI + "certificate-requests/req-1": reply(`{"status":"rejected","errorMessage":"Approval request expired"}`)})
	var diagnostics diag.Diagnostics
	model := fieldCertModel()
	r.pollCertificateRequest(context.Background(), &model, "req-1", 60, &diagnostics)
	if !certificateIssuanceFailed(diagnostics) {
		t.Errorf("a rejected request must stop polling as a failure, got %v", diagnostics)
	}
}

func TestCertDeleteRevokesAndDeletesEveryEarlierRenewal(t *testing.T) {
	setTestNow(t, "2026-10-05T00:00:00Z")
	log := &certRequestLog{}
	r := fakeAPI(t, log, map[string]apiReply{
		"GET " + certAPI + "cert-old":          reply(`{"certificate":{"id":"cert-old","status":"active","caId":"ca-1","notAfter":"2026-10-31T00:00:00Z","renewedFromCertificateId":"cert-prev"}}`),
		"GET " + certAPI + "cert-prev":         reply(`{"certificate":{"id":"cert-prev","status":"active","caId":"ca-1","notAfter":"2026-10-20T00:00:00Z","renewedByCertificateId":"cert-old","renewedFromCertificateId":"cert-first"}}`),
		"GET " + certAPI + "cert-first":        reply(`{"certificate":{"id":"cert-first","status":"active","caId":"ca-1","notAfter":"2026-10-01T00:00:00Z","renewedByCertificateId":"cert-prev"}}`),
		"POST " + certAPI + "cert-old/revoke":  reply(`{"message":"ok"}`),
		"POST " + certAPI + "cert-prev/revoke": reply(`{"message":"ok"}`),
		"DELETE " + certAPI + "cert-old":       reply(`{}`),
		"DELETE " + certAPI + "cert-prev":      reply(`{}`),
		"DELETE " + certAPI + "cert-first":     reply(`{}`),
	})
	state := fieldCertModel()
	state.RevokeOnDestroy = types.BoolValue(true)
	state.DeleteOnDestroy = types.BoolValue(true)
	state.RevocationReason = types.StringValue("key_compromise")
	resp := runDelete(t, r, state)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	want := []string{
		"GET " + certAPI + "cert-old", "GET " + certAPI + "cert-prev", "GET " + certAPI + "cert-first",
		"POST " + certAPI + "cert-prev/revoke", "POST " + certAPI + "cert-old/revoke",
		"DELETE " + certAPI + "cert-first", "DELETE " + certAPI + "cert-prev", "DELETE " + certAPI + "cert-old",
	}
	if strings.Join(log.all(), ",") != strings.Join(want, ",") {
		t.Errorf("requests = %v\nwant revoke of every unexpired generation and delete of all, oldest first", log.all())
	}
}

func TestCertDeleteCancelsPendingRenewalAndRevokesLaterRenewals(t *testing.T) {
	setTestNow(t, "2026-10-05T00:00:00Z")
	log := &certRequestLog{}
	r := fakeAPI(t, log, map[string]apiReply{
		"POST " + certAPI + "certificate-requests/req-slow/cancel": reply(`{"status":"failed","cancelled":true,"errorMessage":null}`),
		"GET " + certAPI + "cert-old":                              reply(`{"certificate":{"id":"cert-old","status":"active","caId":"ca-1","notAfter":"2026-10-31T00:00:00Z","renewedByCertificateId":"cert-next"}}`),
		"GET " + certAPI + "cert-next":                             reply(`{"certificate":{"id":"cert-next","status":"active","caId":"ca-1","notAfter":"2026-11-30T00:00:00Z","renewedFromCertificateId":"cert-old"}}`),
		"POST " + certAPI + "cert-old/revoke":                      reply(`{"message":"ok"}`),
		"POST " + certAPI + "cert-next/revoke":                     reply(`{"message":"ok"}`),
	})
	state := fieldCertModel()
	state.RevokeOnDestroy = types.BoolValue(true)
	s := certTestSchema(t)
	priorState := certState(t, s, &state)
	resp := resource.DeleteResponse{State: priorState}
	value, _ := json.Marshal(pendingRenewal{RequestId: "req-slow"})
	r.delete(context.Background(), resource.DeleteRequest{State: priorState}, fakePrivateState{certificatePendingRenewalKey: value}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	want := []string{
		"POST " + certAPI + "certificate-requests/req-slow/cancel", "GET " + certAPI + "cert-old", "GET " + certAPI + "cert-next",
		"POST " + certAPI + "cert-old/revoke", "POST " + certAPI + "cert-next/revoke",
	}
	if strings.Join(log.all(), ",") != strings.Join(want, ",") {
		t.Errorf("requests = %v, want %v", log.all(), want)
	}
}

func TestCertDeleteRevokeErrorsOnDeletedCertificates(t *testing.T) {
	setTestNow(t, "2026-10-05T00:00:00Z")
	for _, tc := range []struct {
		status       int
		message      string
		wantError    bool
		wantWarnings int
	}{
		{http.StatusInternalServerError, "Something went wrong", true, 0},
		{http.StatusBadRequest, "ADCS gateway operation failed: host unreachable", true, 0},
		{http.StatusBadRequest, "Certificate revocation is not supported by this certificate authority type", false, 1},
	} {
		r := fakeAPI(t, nil, map[string]apiReply{
			"GET " + certAPI + "cert-old":         reply(`{"certificate":{"id":"cert-old","status":"active","caId":"ca-1","notAfter":"2026-10-31T00:00:00Z","deletedAt":"2026-10-04T00:00:00Z"}}`),
			"POST " + certAPI + "cert-old/revoke": {status: tc.status, body: fmt.Sprintf(`{"message":%q}`, tc.message)},
		})
		state := fieldCertModel()
		state.Status = types.StringValue(certificateStatusDeleted)
		state.RevokeOnDestroy = types.BoolValue(true)
		resp := runDelete(t, r, state)
		if resp.Diagnostics.HasError() != tc.wantError || resp.Diagnostics.WarningsCount() != tc.wantWarnings {
			t.Errorf("revoke HTTP %d %q on a deleted certificate: diagnostics = %v, want error %t and %d warnings", tc.status, tc.message, resp.Diagnostics, tc.wantError, tc.wantWarnings)
		}
	}
}
