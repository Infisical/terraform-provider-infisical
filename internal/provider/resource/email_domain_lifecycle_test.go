package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	infisical "terraform-provider-infisical/internal/client"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	emailDomainTestOrgID = "3f1c9c0e-5a7b-4d2e-8f10-1a2b3c4d5e6f"
	emailDomainTestID    = "7c9e6679-7425-40de-944b-e07fc1f90ae7"
	emailDomainTestID2   = "9b2f4c1a-6d3e-4f80-a1b2-c3d4e5f60718"
)

type fakeEmailDomain struct {
	ID                 string            `json:"id"`
	OrgID              string            `json:"orgId"`
	Domain             string            `json:"domain"`
	VerificationRecord map[string]string `json:"verificationRecord"`
	Status             string            `json:"status"`
}

type fakeEmailDomainBackend struct {
	t                  *testing.T
	nextEmailDomainIDs []string
	emailDomains       []*fakeEmailDomain
	// Set when the organization enforces SSO: the service then refuses to delete the only verified domain.
	authEnforced bool
	forbidList   bool
	requests     []string
	bodies       []map[string]any
}

func newFakeEmailDomainBackend(t *testing.T) *fakeEmailDomainBackend {
	return &fakeEmailDomainBackend{
		t:                  t,
		nextEmailDomainIDs: []string{emailDomainTestID, emailDomainTestID2},
	}
}

func (b *fakeEmailDomainBackend) writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		b.t.Error(err)
	}
}

func (b *fakeEmailDomainBackend) apiError(w http.ResponseWriter, status int, message string) {
	b.writeJSON(w, status, map[string]any{"statusCode": status, "message": message})
}

func (b *fakeEmailDomainBackend) requestsMatching(method string) int {
	count := 0
	for _, request := range b.requests {
		if strings.HasPrefix(request, method+" ") {
			count++
		}
	}
	return count
}

func (b *fakeEmailDomainBackend) findEmailDomain(id string) (int, *fakeEmailDomain) {
	for i, emailDomain := range b.emailDomains {
		if emailDomain.ID == id {
			return i, emailDomain
		}
	}
	return -1, nil
}

func (b *fakeEmailDomainBackend) verifiedEmailDomains() int {
	count := 0
	for _, emailDomain := range b.emailDomains {
		if emailDomain.Status == "verified" {
			count++
		}
	}
	return count
}

func (b *fakeEmailDomainBackend) handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/email-domains", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			b.apiError(w, http.StatusUnprocessableEntity, "Invalid body")
			return
		}
		b.bodies = append(b.bodies, body)

		rawDomain, _ := body["domain"].(string)

		domain := strings.ToLower(strings.TrimSpace(rawDomain))
		if domain == "" {
			b.apiError(w, http.StatusUnprocessableEntity, "Invalid body")
			return
		}
		for _, existing := range b.emailDomains {
			if existing.Domain != domain {
				continue
			}
			if existing.Status == "verified" {
				b.apiError(w, http.StatusBadRequest, "This domain is already verified by an organization.")
				return
			}
			b.apiError(w, http.StatusBadRequest, "This domain is already pending verification or has expired.")
			return
		}
		if len(b.nextEmailDomainIDs) == 0 {
			b.t.Error("the fake backend has no email domain IDs left")
			b.apiError(w, http.StatusInternalServerError, "no IDs left")
			return
		}

		emailDomain := &fakeEmailDomain{
			ID:     b.nextEmailDomainIDs[0],
			OrgID:  emailDomainTestOrgID,
			Domain: domain,
			VerificationRecord: map[string]string{
				"type":  "TXT",
				"name":  "_infisical-verification." + domain,
				"value": fmt.Sprintf("infisical-domain-verification=code-%d", len(b.emailDomains)+1),
			},
			Status: "pending",
		}
		b.nextEmailDomainIDs = b.nextEmailDomainIDs[1:]
		b.emailDomains = append(b.emailDomains, emailDomain)
		b.writeJSON(w, http.StatusOK, map[string]any{"emailDomain": emailDomain})
	})

	mux.HandleFunc("GET /api/v1/email-domains", func(w http.ResponseWriter, _ *http.Request) {
		if b.forbidList {
			b.apiError(w, http.StatusForbidden, "You are not allowed to read email domains")
			return
		}
		emailDomains := b.emailDomains
		if emailDomains == nil {
			emailDomains = []*fakeEmailDomain{}
		}
		b.writeJSON(w, http.StatusOK, map[string]any{"emailDomains": emailDomains})
	})

	mux.HandleFunc("DELETE /api/v1/email-domains/{id}", func(w http.ResponseWriter, r *http.Request) {
		index, emailDomain := b.findEmailDomain(r.PathValue("id"))
		if emailDomain == nil {
			b.apiError(w, http.StatusNotFound, "Email domain not found.")
			return
		}
		if emailDomain.Status == "verified" && b.authEnforced && b.verifiedEmailDomains() <= 1 {
			b.apiError(w, http.StatusBadRequest, fmt.Sprintf("Email domain '%s' is the organization's only verified domain and SSO enforcement is on. Turn off SSO enforcement before deleting it.", emailDomain.Domain))
			return
		}
		b.emailDomains = append(b.emailDomains[:index], b.emailDomains[index+1:]...)
		b.writeJSON(w, http.StatusOK, map[string]any{"emailDomain": emailDomain})
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b.requests = append(b.requests, r.Method+" "+r.URL.Path)
		mux.ServeHTTP(w, r)
	})
}

func newFakeEmailDomainClient(t *testing.T, backend *fakeEmailDomainBackend) *infisical.Client {
	t.Helper()
	server := httptest.NewServer(backend.handler())
	t.Cleanup(server.Close)
	return &infisical.Client{Config: infisical.Config{
		HttpClient:            resty.New().SetBaseURL(server.URL),
		IsMachineIdentityAuth: true,
	}}
}

func emailDomainTestSchema(t *testing.T) schema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	NewEmailDomainResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func emailDomainCreatePlan(t *testing.T, s schema.Schema, domain string) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: s}
	if diags := plan.Set(context.Background(), &EmailDomainResourceModel{
		ID:                 types.StringUnknown(),
		Domain:             types.StringValue(domain),
		VerificationRecord: types.ObjectUnknown(emailDomainVerificationRecordAttrTypes),
	}); diags.HasError() {
		t.Fatal(diags)
	}
	return plan
}

func emailDomainState(t *testing.T, s schema.Schema, model EmailDomainResourceModel) tfsdk.State {
	t.Helper()
	if len(model.VerificationRecord.AttributeTypes(context.Background())) == 0 {
		model.VerificationRecord = types.ObjectNull(emailDomainVerificationRecordAttrTypes)
	}
	state := tfsdk.State{Schema: s}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return state
}

func emailDomainModel(t *testing.T, state tfsdk.State) EmailDomainResourceModel {
	t.Helper()
	var model EmailDomainResourceModel
	if diags := state.Get(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	return model
}

func createEmailDomain(r *EmailDomainResource, plan tfsdk.Plan) resource.CreateResponse {
	resp := resource.CreateResponse{State: tfsdk.State{Schema: plan.Schema}}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, &resp)
	return resp
}

func mustCreateEmailDomain(t *testing.T, r *EmailDomainResource, plan tfsdk.Plan) tfsdk.State {
	t.Helper()
	resp := createEmailDomain(r, plan)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	return resp.State
}

func readEmailDomain(r *EmailDomainResource, state tfsdk.State) resource.ReadResponse {
	resp := resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
	return resp
}

func deleteEmailDomain(r *EmailDomainResource, state tfsdk.State) resource.DeleteResponse {
	resp := resource.DeleteResponse{State: state}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, &resp)
	return resp
}

func TestEmailDomainLifecycle(t *testing.T) {
	s := emailDomainTestSchema(t)
	backend := newFakeEmailDomainBackend(t)
	r := &EmailDomainResource{client: newFakeEmailDomainClient(t, backend)}

	state := mustCreateEmailDomain(t, r, emailDomainCreatePlan(t, s, "example.com"))

	if backend.requests[0] != "POST /api/v1/email-domains" {
		t.Errorf("unexpected first request: %q", backend.requests[0])
	}
	if body := backend.bodies[0]; len(body) != 1 || body["domain"] != "example.com" {
		t.Errorf("unexpected create body: %v", body)
	}

	created := emailDomainModel(t, state)
	want := EmailDomainResourceModel{
		ID:     types.StringValue(emailDomainTestID),
		Domain: types.StringValue("example.com"),
		VerificationRecord: types.ObjectValueMust(emailDomainVerificationRecordAttrTypes, map[string]attr.Value{
			"type":  types.StringValue("TXT"),
			"name":  types.StringValue("_infisical-verification.example.com"),
			"value": types.StringValue("infisical-domain-verification=code-1"),
		}),
	}
	if !reflect.DeepEqual(created, want) {
		t.Errorf("unexpected state after create: %+v", created)
	}

	readResp := readEmailDomain(r, state)
	if readResp.Diagnostics.HasError() {
		t.Fatal(readResp.Diagnostics)
	}
	if refreshed := emailDomainModel(t, readResp.State); !reflect.DeepEqual(refreshed, created) {
		t.Errorf("expected no drift after a refresh, got %+v, want %+v", refreshed, created)
	}

	if resp := deleteEmailDomain(r, state); resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if len(backend.emailDomains) != 0 {
		t.Error("expected the email domain to be deleted on the backend")
	}
	if want := "DELETE /api/v1/email-domains/" + emailDomainTestID; backend.requests[len(backend.requests)-1] != want {
		t.Errorf("expected the last request to be %q, got %q", want, backend.requests[len(backend.requests)-1])
	}
}

func TestEmailDomainReadPicksItsOwnDomain(t *testing.T) {
	s := emailDomainTestSchema(t)
	backend := newFakeEmailDomainBackend(t)
	r := &EmailDomainResource{client: newFakeEmailDomainClient(t, backend)}

	first := mustCreateEmailDomain(t, r, emailDomainCreatePlan(t, s, "example.com"))
	second := mustCreateEmailDomain(t, r, emailDomainCreatePlan(t, s, "example.org"))

	for domain, state := range map[string]tfsdk.State{"example.com": first, "example.org": second} {
		resp := readEmailDomain(r, state)
		if resp.Diagnostics.HasError() {
			t.Fatal(resp.Diagnostics)
		}
		if got := emailDomainModel(t, resp.State).Domain.ValueString(); got != domain {
			t.Errorf("expected %q, got %q", domain, got)
		}
	}
}

func TestEmailDomainCreateAlreadyClaimed(t *testing.T) {
	s := emailDomainTestSchema(t)
	backend := newFakeEmailDomainBackend(t)
	r := &EmailDomainResource{client: newFakeEmailDomainClient(t, backend)}

	mustCreateEmailDomain(t, r, emailDomainCreatePlan(t, s, "example.com"))

	resp := createEmailDomain(r, emailDomainCreatePlan(t, s, "example.com"))
	if !resp.Diagnostics.HasError() || !strings.Contains(fmt.Sprint(resp.Diagnostics), "already pending verification") {
		t.Errorf("expected the API's conflict message, got: %v", resp.Diagnostics)
	}
}

func TestEmailDomainReadDropsDeletedDomain(t *testing.T) {
	s := emailDomainTestSchema(t)
	backend := newFakeEmailDomainBackend(t)
	r := &EmailDomainResource{client: newFakeEmailDomainClient(t, backend)}

	state := mustCreateEmailDomain(t, r, emailDomainCreatePlan(t, s, "example.com"))
	backend.emailDomains = nil

	resp := readEmailDomain(r, state)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Error("expected the resource to be removed from state")
	}
}

// A failed list is an error. It must not remove the resource from state.
func TestEmailDomainReadListError(t *testing.T) {
	s := emailDomainTestSchema(t)
	backend := newFakeEmailDomainBackend(t)
	r := &EmailDomainResource{client: newFakeEmailDomainClient(t, backend)}

	state := mustCreateEmailDomain(t, r, emailDomainCreatePlan(t, s, "example.com"))
	backend.forbidList = true

	resp := readEmailDomain(r, state)
	if !resp.Diagnostics.HasError() {
		t.Error("expected an error")
	}
	if resp.State.Raw.IsNull() {
		t.Error("expected the resource to stay in state")
	}
}

func TestEmailDomainReadEmptyID(t *testing.T) {
	backend := newFakeEmailDomainBackend(t)
	r := &EmailDomainResource{client: newFakeEmailDomainClient(t, backend)}

	state := emailDomainState(t, emailDomainTestSchema(t), EmailDomainResourceModel{
		ID:     types.StringValue(""),
		Domain: types.StringValue("example.com"),
	})
	resp := readEmailDomain(r, state)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Error("expected the resource to be removed from state")
	}
	if len(backend.requests) != 0 {
		t.Errorf("expected no API call, got %v", backend.requests)
	}
}

func TestEmailDomainDelete(t *testing.T) {
	s := emailDomainTestSchema(t)

	for name, tc := range map[string]struct {
		prepare   func(b *fakeEmailDomainBackend)
		wantError string
	}{
		"already deleted": {prepare: func(b *fakeEmailDomainBackend) { b.emailDomains = nil }},
		"only verified domain while SSO is enforced": {
			prepare: func(b *fakeEmailDomainBackend) {
				b.emailDomains[0].Status = "verified"
				b.authEnforced = true
			},
			wantError: "Turn off SSO enforcement",
		},
	} {
		t.Run(name, func(t *testing.T) {
			backend := newFakeEmailDomainBackend(t)
			r := &EmailDomainResource{client: newFakeEmailDomainClient(t, backend)}

			state := mustCreateEmailDomain(t, r, emailDomainCreatePlan(t, s, "example.com"))
			tc.prepare(backend)

			resp := deleteEmailDomain(r, state)
			if tc.wantError == "" {
				if resp.Diagnostics.HasError() {
					t.Errorf("expected a 404 to count as deleted, got: %v", resp.Diagnostics)
				}
			} else if !strings.Contains(fmt.Sprint(resp.Diagnostics), tc.wantError) {
				t.Errorf("expected an error with %q, got: %v", tc.wantError, resp.Diagnostics)
			}
			if backend.requestsMatching(http.MethodDelete) != 1 {
				t.Errorf("expected one DELETE, requests: %v", backend.requests)
			}
		})
	}
}

func TestEmailDomainImport(t *testing.T) {
	ctx := context.Background()
	s := emailDomainTestSchema(t)
	backend := newFakeEmailDomainBackend(t)
	r := &EmailDomainResource{client: newFakeEmailDomainClient(t, backend)}

	created := emailDomainModel(t, mustCreateEmailDomain(t, r, emailDomainCreatePlan(t, s, "example.com")))

	invalid := resource.ImportStateResponse{State: emailDomainState(t, s, EmailDomainResourceModel{})}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "example.com"}, &invalid)
	if !invalid.Diagnostics.HasError() || !strings.Contains(fmt.Sprint(invalid.Diagnostics), "Invalid import ID") {
		t.Errorf("expected an Invalid import ID error, got: %v", invalid.Diagnostics)
	}

	resp := resource.ImportStateResponse{State: emailDomainState(t, s, EmailDomainResourceModel{})}
	r.ImportState(ctx, resource.ImportStateRequest{ID: emailDomainTestID}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}

	readResp := readEmailDomain(r, resp.State)
	if readResp.Diagnostics.HasError() {
		t.Fatal(readResp.Diagnostics)
	}
	if imported := emailDomainModel(t, readResp.State); !reflect.DeepEqual(imported, created) {
		t.Errorf("expected the import to rebuild the full state, got %+v, want %+v", imported, created)
	}
}

func TestEmailDomainSchemaRejectsUnnormalizedDomain(t *testing.T) {
	s := emailDomainTestSchema(t)
	attribute, ok := s.Attributes["domain"].(schema.StringAttribute)
	if !ok {
		t.Fatal("domain is not a string attribute")
	}

	for value, wantError := range map[string]bool{
		"example.com":     false,
		"sub.example.com": false,
		"Example.com":     true,
		" example.com":    true,
		"example.com ":    true,
	} {
		for _, v := range attribute.Validators {
			req := validator.StringRequest{Path: path.Root("domain"), ConfigValue: types.StringValue(value)}
			var resp validator.StringResponse
			v.ValidateString(context.Background(), req, &resp)
			if resp.Diagnostics.HasError() != wantError {
				t.Errorf("domain %q: wantError=%t, got: %v", value, wantError, resp.Diagnostics)
			}
		}
	}
}
