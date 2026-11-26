package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const emailDomainVerificationTestID = "5d0a7b1e-3c2f-4e6a-9b8d-1f2e3d4c5b6a"

type fakeEmailDomainVerificationBackend struct {
	t        *testing.T
	statuses map[string]string
	// The number of verify calls that miss the DNS record before it is found.
	dnsMisses      int
	planRestricted bool
	// Another organization verifies the domain while this one is pending, so the final claim is refused.
	verifiedByAnotherOrg bool
	// Sends errors without the `error` code, as an older Infisical does.
	omitErrorCodes bool
	requests       []string
}

func newFakeEmailDomainVerificationBackend(t *testing.T) *fakeEmailDomainVerificationBackend {
	return &fakeEmailDomainVerificationBackend{
		t:        t,
		statuses: map[string]string{emailDomainVerificationTestID: "pending"},
	}
}

func (b *fakeEmailDomainVerificationBackend) writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		b.t.Error(err)
	}
}

func (b *fakeEmailDomainVerificationBackend) apiError(w http.ResponseWriter, status int, code, message string) {
	body := map[string]any{"statusCode": status, "message": message}
	if !b.omitErrorCodes {
		body["error"] = code
	}
	b.writeJSON(w, status, body)
}

func (b *fakeEmailDomainVerificationBackend) emailDomain(id string) map[string]any {
	return map[string]any{
		"id":                     id,
		"domain":                 "example.com",
		"verificationCode":       "infisical-domain-verification=code",
		"verificationRecordName": "_infisical-verification.example.com",
		"status":                 b.statuses[id],
	}
}

func (b *fakeEmailDomainVerificationBackend) handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/email-domains", func(w http.ResponseWriter, _ *http.Request) {
		emailDomains := []map[string]any{}
		for id := range b.statuses {
			emailDomains = append(emailDomains, b.emailDomain(id))
		}
		b.writeJSON(w, http.StatusOK, map[string]any{"emailDomains": emailDomains})
	})

	mux.HandleFunc("POST /api/v1/email-domains/{id}/verify", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if b.planRestricted {
			b.apiError(w, http.StatusBadRequest, "EMAIL_DOMAIN_PLAN_RESTRICTED", "Failed to verify email domain due to plan restriction. Upgrade plan to use email domain verification.")
			return
		}
		status, ok := b.statuses[id]
		if !ok {
			b.apiError(w, http.StatusNotFound, "NotFound", "Email domain not found.")
			return
		}
		switch status {
		case "verified":
			b.apiError(w, http.StatusConflict, "Conflict", "Email domain 'example.com' is already verified.")
			return
		case "expired":
			b.apiError(w, http.StatusBadRequest, "EMAIL_DOMAIN_VERIFICATION_EXPIRED", "Verification code has expired. Please create a new domain verification request.")
			return
		}
		if b.dnsMisses > 0 {
			b.dnsMisses--
			// The service answers with one of two messages, depending on whether the lookup itself failed.
			if b.dnsMisses%2 == 0 {
				b.apiError(w, http.StatusBadRequest, "EMAIL_DOMAIN_DNS_LOOKUP_FAILED", "Failed to resolve DNS records for the domain. Please verify the DNS TXT record exists and try again. DNS changes may take time to propagate.")
			} else {
				b.apiError(w, http.StatusBadRequest, "EMAIL_DOMAIN_DNS_RECORD_MISMATCH", "DNS TXT record not found or value does not match. Expected a TXT record at _infisical-verification.example.com with value: infisical-domain-verification=code")
			}
			return
		}
		if b.verifiedByAnotherOrg {
			b.apiError(w, http.StatusBadRequest, "EMAIL_DOMAIN_ALREADY_VERIFIED", "This domain is already verified by an organization.")
			return
		}
		b.statuses[id] = "verified"
		b.writeJSON(w, http.StatusOK, map[string]any{"emailDomain": b.emailDomain(id)})
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b.requests = append(b.requests, r.Method+" "+r.URL.Path)
		mux.ServeHTTP(w, r)
	})
}

func (b *fakeEmailDomainVerificationBackend) verifyRequests() int {
	count := 0
	for _, request := range b.requests {
		if strings.HasSuffix(request, "/verify") {
			count++
		}
	}
	return count
}

func newEmailDomainVerificationTestResource(t *testing.T, backend *fakeEmailDomainVerificationBackend) *EmailDomainVerificationResource {
	t.Helper()
	server := httptest.NewServer(backend.handler())
	t.Cleanup(server.Close)
	useFastEmailDomainVerificationPolling(t)
	return &EmailDomainVerificationResource{
		client: &infisical.Client{Config: infisical.Config{
			HttpClient:            resty.New().SetBaseURL(server.URL),
			IsMachineIdentityAuth: true,
		}},
	}
}

// useFastEmailDomainVerificationPolling sets both retry intervals to 1ms so retry tests finish quickly.
func useFastEmailDomainVerificationPolling(t *testing.T) {
	t.Helper()
	initial, maxInterval := emailDomainVerificationInitialInterval, emailDomainVerificationMaxInterval
	emailDomainVerificationInitialInterval, emailDomainVerificationMaxInterval = time.Millisecond, time.Millisecond
	t.Cleanup(func() {
		emailDomainVerificationInitialInterval, emailDomainVerificationMaxInterval = initial, maxInterval
	})
}

func emailDomainVerificationTestSchema(t *testing.T) schema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	NewEmailDomainVerificationResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

// emailDomainVerificationTimeouts returns the timeouts block, null when create is empty.
func emailDomainVerificationTimeouts(create string) timeouts.Value {
	attrTypes := map[string]attr.Type{"create": types.StringType}
	if create == "" {
		return timeouts.Value{Object: types.ObjectNull(attrTypes)}
	}
	return timeouts.Value{Object: types.ObjectValueMust(attrTypes, map[string]attr.Value{"create": types.StringValue(create)})}
}

func emailDomainVerificationPlan(t *testing.T, s schema.Schema, createTimeout string) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: s}
	if diags := plan.Set(context.Background(), &EmailDomainVerificationResourceModel{
		ID:            types.StringUnknown(),
		EmailDomainID: types.StringValue(emailDomainVerificationTestID),
		Timeouts:      emailDomainVerificationTimeouts(createTimeout),
	}); diags.HasError() {
		t.Fatal(diags)
	}
	return plan
}

func emailDomainVerificationState(t *testing.T, s schema.Schema, model EmailDomainVerificationResourceModel) tfsdk.State {
	t.Helper()
	if len(model.Timeouts.Object.AttributeTypes(context.Background())) == 0 {
		model.Timeouts = emailDomainVerificationTimeouts("")
	}
	state := tfsdk.State{Schema: s}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return state
}

func emailDomainVerificationModel(t *testing.T, state tfsdk.State) EmailDomainVerificationResourceModel {
	t.Helper()
	var model EmailDomainVerificationResourceModel
	if diags := state.Get(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return model
}

func createEmailDomainVerification(r *EmailDomainVerificationResource, plan tfsdk.Plan) resource.CreateResponse {
	resp := resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema}}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, &resp)
	return resp
}

func readEmailDomainVerification(r *EmailDomainVerificationResource, state tfsdk.State) resource.ReadResponse {
	resp := resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
	return resp
}

func TestEmailDomainVerificationLifecycle(t *testing.T) {
	s := emailDomainVerificationTestSchema(t)
	backend := newFakeEmailDomainVerificationBackend(t)
	// One miss of each kind: the record is not published yet, then not propagated yet.
	backend.dnsMisses = 2
	r := newEmailDomainVerificationTestResource(t, backend)

	resp := createEmailDomainVerification(r, emailDomainVerificationPlan(t, s, ""))
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if got := backend.verifyRequests(); got != 3 {
		t.Errorf("expected 3 verify requests (2 DNS misses, then success), got %d", got)
	}
	created := emailDomainVerificationModel(t, resp.State)
	if created.ID.ValueString() != emailDomainVerificationTestID || created.EmailDomainID.ValueString() != emailDomainVerificationTestID {
		t.Errorf("unexpected state after create: %+v", created)
	}

	readResp := readEmailDomainVerification(r, resp.State)
	if readResp.Diagnostics.HasError() {
		t.Fatal(readResp.Diagnostics)
	}
	if refreshed := emailDomainVerificationModel(t, readResp.State); refreshed.ID != created.ID || refreshed.EmailDomainID != created.EmailDomainID {
		t.Errorf("expected no drift after a refresh, got %+v, want %+v", refreshed, created)
	}

	requestsBefore := len(backend.requests)
	var deleteResp resource.DeleteResponse
	r.Delete(context.Background(), resource.DeleteRequest{State: resp.State}, &deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatal(deleteResp.Diagnostics)
	}
	if len(backend.requests) != requestsBefore {
		t.Errorf("expected delete to make no API call, got %v", backend.requests[requestsBefore:])
	}
}

func TestEmailDomainVerificationCreateAlreadyVerified(t *testing.T) {
	s := emailDomainVerificationTestSchema(t)
	backend := newFakeEmailDomainVerificationBackend(t)
	backend.statuses[emailDomainVerificationTestID] = "verified"
	r := newEmailDomainVerificationTestResource(t, backend)

	resp := createEmailDomainVerification(r, emailDomainVerificationPlan(t, s, ""))
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected the 409 for an already verified domain to count as success, got: %v", resp.Diagnostics)
	}
}

// Errors that a retry cannot fix stop at the first attempt.
func TestEmailDomainVerificationCreateFailsAtOnce(t *testing.T) {
	for name, tc := range map[string]struct {
		setup func(b *fakeEmailDomainVerificationBackend)
		want  string
	}{
		"expired code": {
			setup: func(b *fakeEmailDomainVerificationBackend) { b.statuses[emailDomainVerificationTestID] = "expired" },
			want:  "Replace the infisical_email_domain",
		},
		"plan restriction": {
			setup: func(b *fakeEmailDomainVerificationBackend) { b.planRestricted = true },
			want:  "plan restriction",
		},
		"unknown domain": {
			setup: func(b *fakeEmailDomainVerificationBackend) { delete(b.statuses, emailDomainVerificationTestID) },
			want:  "was not found",
		},
		"domain verified by another organization": {
			setup: func(b *fakeEmailDomainVerificationBackend) { b.verifiedByAnotherOrg = true },
			want:  "Another organization has already verified this domain",
		},
		// The retry follows the error code, not the message text.
		"DNS miss without an error code": {
			setup: func(b *fakeEmailDomainVerificationBackend) {
				b.dnsMisses = 1
				b.omitErrorCodes = true
			},
			want: "Failed to resolve DNS records",
		},
	} {
		t.Run(name, func(t *testing.T) {
			s := emailDomainVerificationTestSchema(t)
			backend := newFakeEmailDomainVerificationBackend(t)
			tc.setup(backend)
			r := newEmailDomainVerificationTestResource(t, backend)

			resp := createEmailDomainVerification(r, emailDomainVerificationPlan(t, s, ""))
			if !resp.Diagnostics.HasError() || !strings.Contains(fmt.Sprint(resp.Diagnostics), tc.want) {
				t.Errorf("expected an error containing %q, got: %v", tc.want, resp.Diagnostics)
			}
			if got := backend.verifyRequests(); got != 1 {
				t.Errorf("expected 1 verify request, got %d", got)
			}
		})
	}
}

func TestEmailDomainVerificationCreateTimesOut(t *testing.T) {
	s := emailDomainVerificationTestSchema(t)
	backend := newFakeEmailDomainVerificationBackend(t)
	backend.dnsMisses = 1_000_000
	r := newEmailDomainVerificationTestResource(t, backend)

	resp := createEmailDomainVerification(r, emailDomainVerificationPlan(t, s, "50ms"))
	diags := fmt.Sprint(resp.Diagnostics)
	if !resp.Diagnostics.HasError() || !strings.Contains(diags, "timed out") || !strings.Contains(diags, "DNS") {
		t.Errorf("expected a timeout error with the last DNS message, got: %v", resp.Diagnostics)
	}
	if backend.verifyRequests() < 2 {
		t.Errorf("expected the verification to be retried, got %d requests", backend.verifyRequests())
	}
}

func TestEmailDomainVerificationReadDropsUnverifiedDomain(t *testing.T) {
	for name, change := range map[string]func(b *fakeEmailDomainVerificationBackend){
		"domain deleted":      func(b *fakeEmailDomainVerificationBackend) { delete(b.statuses, emailDomainVerificationTestID) },
		"domain not verified": func(b *fakeEmailDomainVerificationBackend) { b.statuses[emailDomainVerificationTestID] = "pending" },
	} {
		t.Run(name, func(t *testing.T) {
			s := emailDomainVerificationTestSchema(t)
			backend := newFakeEmailDomainVerificationBackend(t)
			r := newEmailDomainVerificationTestResource(t, backend)

			resp := createEmailDomainVerification(r, emailDomainVerificationPlan(t, s, ""))
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			change(backend)

			readResp := readEmailDomainVerification(r, resp.State)
			if readResp.Diagnostics.HasError() {
				t.Fatal(readResp.Diagnostics)
			}
			if !readResp.State.Raw.IsNull() {
				t.Error("expected the resource to be removed from state")
			}
		})
	}
}

func TestEmailDomainVerificationReadEmptyID(t *testing.T) {
	backend := newFakeEmailDomainVerificationBackend(t)
	r := newEmailDomainVerificationTestResource(t, backend)

	state := emailDomainVerificationState(t, emailDomainVerificationTestSchema(t), EmailDomainVerificationResourceModel{
		ID:            types.StringValue(""),
		EmailDomainID: types.StringValue(emailDomainVerificationTestID),
	})
	if readResp := readEmailDomainVerification(r, state); !readResp.State.Raw.IsNull() {
		t.Error("expected the resource to be removed from state")
	}
	if len(backend.requests) != 0 {
		t.Errorf("expected no API call, got %v", backend.requests)
	}
}

func TestEmailDomainVerificationImport(t *testing.T) {
	ctx := context.Background()
	s := emailDomainVerificationTestSchema(t)
	backend := newFakeEmailDomainVerificationBackend(t)
	backend.statuses[emailDomainVerificationTestID] = "verified"
	r := newEmailDomainVerificationTestResource(t, backend)

	invalid := resource.ImportStateResponse{State: emailDomainVerificationState(t, s, EmailDomainVerificationResourceModel{})}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "example.com"}, &invalid)
	if !invalid.Diagnostics.HasError() || !strings.Contains(fmt.Sprint(invalid.Diagnostics), "Invalid import ID") {
		t.Errorf("expected an Invalid import ID error, got: %v", invalid.Diagnostics)
	}

	resp := resource.ImportStateResponse{State: emailDomainVerificationState(t, s, EmailDomainVerificationResourceModel{})}
	r.ImportState(ctx, resource.ImportStateRequest{ID: emailDomainVerificationTestID}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}

	readResp := readEmailDomainVerification(r, resp.State)
	if readResp.Diagnostics.HasError() {
		t.Fatal(readResp.Diagnostics)
	}
	imported := emailDomainVerificationModel(t, readResp.State)
	if imported.ID.ValueString() != emailDomainVerificationTestID || imported.EmailDomainID.ValueString() != emailDomainVerificationTestID {
		t.Errorf("expected the import to rebuild the full state, got %+v", imported)
	}
	if backend.verifyRequests() != 0 {
		t.Error("expected import to read only, not verify")
	}
}
