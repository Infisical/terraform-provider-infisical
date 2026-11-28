package infisicalclient

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	infisicalerrors "terraform-provider-infisical/internal/errors"
)

const emailDomainsListJSON = `{"emailDomains":[
	{"id":"d1","orgId":"o1","domain":"example.com","verificationRecord":{"type":"TXT","name":"_infisical-verification.example.com","value":"infisical-domain-verification=code1"},"status":"pending"},
	{"id":"d2","orgId":"o1","domain":"example.org","verificationRecord":{"type":"TXT","name":"_infisical-verification.example.org","value":"infisical-domain-verification=code2"},"status":"verified"}
]}`

func TestCreateEmailDomainRequest(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		if err := json.Unmarshal(raw, &gotBody); err != nil {
			t.Error(err)
		}
		jsonResponse(http.StatusOK, `{"emailDomain":{"id":"d1","domain":"example.com","verificationRecord":{"type":"TXT","name":"_infisical-verification.example.com","value":"infisical-domain-verification=code1"}}}`)(w, r)
	})

	emailDomain, err := client.CreateEmailDomain(CreateEmailDomainRequest{Domain: "example.com"})
	if err != nil {
		t.Fatal(err)
	}

	if gotMethod != http.MethodPost || gotPath != "/api/v1/email-domains" {
		t.Errorf("unexpected request: %s %s", gotMethod, gotPath)
	}
	if len(gotBody) != 1 || gotBody["domain"] != "example.com" {
		t.Errorf("unexpected request body: %v", gotBody)
	}
	want := EmailDomain{ID: "d1", Domain: "example.com", VerificationRecord: EmailDomainVerificationRecord{Type: "TXT", Name: "_infisical-verification.example.com", Value: "infisical-domain-verification=code1"}}
	if emailDomain != want {
		t.Errorf("unexpected email domain: %+v", emailDomain)
	}
}

// The API has no get by ID endpoint, so the client finds the ID in the list.
func TestGetEmailDomainByIDFiltersList(t *testing.T) {
	client := testClient(t, jsonResponse(http.StatusOK, emailDomainsListJSON))

	emailDomain, err := client.GetEmailDomainByID(GetEmailDomainByIDRequest{ID: "d2"})
	if err != nil {
		t.Fatal(err)
	}
	if emailDomain.Domain != "example.org" || emailDomain.VerificationRecord.Value != "infisical-domain-verification=code2" {
		t.Errorf("unexpected email domain: %+v", emailDomain)
	}

	if _, err := client.GetEmailDomainByID(GetEmailDomainByIDRequest{ID: "missing"}); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound for an ID that is not in the list, got: %v", err)
	}
}

// A failed list must not look like a missing domain, or Read would drop the resource from state.
func TestGetEmailDomainByIDListError(t *testing.T) {
	client := testClient(t, jsonResponse(http.StatusForbidden, `{"statusCode":403,"message":"You are not allowed to read email domains"}`))

	if _, err := client.GetEmailDomainByID(GetEmailDomainByIDRequest{ID: "d1"}); err == nil || errors.Is(err, ErrNotFound) {
		t.Errorf("expected a non-ErrNotFound error, got: %v", err)
	}
}

func TestDeleteEmailDomain(t *testing.T) {
	var gotMethod, gotPath string
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		jsonResponse(http.StatusOK, `{"emailDomain":{"id":"d1","domain":"example.com"}}`)(w, r)
	})

	if err := client.DeleteEmailDomain(DeleteEmailDomainRequest{ID: "d1"}); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/api/v1/email-domains/d1" {
		t.Errorf("unexpected request: %s %s", gotMethod, gotPath)
	}

	notFound := testClient(t, jsonResponse(http.StatusNotFound, `{"statusCode":404,"message":"Email domain not found."}`))
	if err := notFound.DeleteEmailDomain(DeleteEmailDomainRequest{ID: "d1"}); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}

	// The API answers 400 for the only verified domain of an organization that enforces SSO.
	blocked := testClient(t, jsonResponse(http.StatusBadRequest, `{"statusCode":400,"message":"Turn off SSO enforcement before deleting it."}`))
	if err := blocked.DeleteEmailDomain(DeleteEmailDomainRequest{ID: "d1"}); err == nil || errors.Is(err, ErrNotFound) {
		t.Errorf("expected a non-ErrNotFound error, got: %v", err)
	}
}

func TestVerifyEmailDomain(t *testing.T) {
	var gotMethod, gotPath string
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		jsonResponse(http.StatusOK, `{"emailDomain":{"id":"d1","domain":"example.com","status":"verified"}}`)(w, r)
	})

	emailDomain, err := client.VerifyEmailDomainWithContext(context.Background(), VerifyEmailDomainRequest{ID: "d1"})
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/v1/email-domains/d1/verify" {
		t.Errorf("unexpected request: %s %s", gotMethod, gotPath)
	}
	if emailDomain.Status != EmailDomainStatusVerified {
		t.Errorf("unexpected email domain: %+v", emailDomain)
	}

	notFound := testClient(t, jsonResponse(http.StatusNotFound, `{"statusCode":404,"message":"Email domain not found."}`))
	if _, err := notFound.VerifyEmailDomainWithContext(context.Background(), VerifyEmailDomainRequest{ID: "d1"}); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}

	conflict := testClient(t, jsonResponse(http.StatusConflict, `{"statusCode":409,"message":"Email domain 'example.com' is already verified."}`))
	_, err = conflict.VerifyEmailDomainWithContext(context.Background(), VerifyEmailDomainRequest{ID: "d1"})
	var apiErr *infisicalerrors.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict || apiErr.ErrorMessage != "Email domain 'example.com' is already verified." {
		t.Errorf("expected a 409 APIError with the message, got: %v", err)
	}

	dnsMiss := testClient(t, jsonResponse(http.StatusBadRequest, `{"statusCode":400,"error":"EMAIL_DOMAIN_DNS_RECORD_MISMATCH","message":"DNS TXT record not found or value does not match."}`))
	_, err = dnsMiss.VerifyEmailDomainWithContext(context.Background(), VerifyEmailDomainRequest{ID: "d1"})
	if !errors.As(err, &apiErr) || apiErr.ErrorCode != "EMAIL_DOMAIN_DNS_RECORD_MISMATCH" {
		t.Errorf("expected a 400 APIError with the error code, got: %v", err)
	}
}
