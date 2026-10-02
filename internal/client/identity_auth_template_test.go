package infisicalclient

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
)

// identityAuthTemplateServer stands in for the identity-templates API. Retries are deliberately not
// configured, so a 5xx surfaces as the error it is.
func identityAuthTemplateServer(t *testing.T, handler http.HandlerFunc) Client {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	return Client{Config: Config{
		HostURL:               srv.URL,
		HttpClient:            resty.New().SetBaseURL(srv.URL),
		IsMachineIdentityAuth: true,
	}}
}

// capturedRequest records the method, path, query and decoded body of the one request a test sends.
type capturedRequest struct {
	Method string
	Path   string
	Query  string
	Body   map[string]any
}

func capturingHandler(t *testing.T, captured *capturedRequest, response string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		captured.Method = r.Method
		captured.Path = r.URL.Path
		captured.Query = r.URL.RawQuery
		raw, _ := io.ReadAll(r.Body)
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &captured.Body); err != nil {
				t.Errorf("request body is not a JSON object: %s", raw)
			}
		}
		jsonResponse(http.StatusOK, response)(w, r)
	}
}

const kubernetesTemplateResponse = `{
	"id": "11111111-1111-1111-1111-111111111111",
	"name": "prod-cluster",
	"orgId": "99999999-9999-9999-9999-999999999999",
	"authMethod": "kubernetes",
	"templateFields": {
		"tokenReviewMode": "gateway",
		"kubernetesHost": null,
		"caCert": "",
		"verifyTlsCertificate": false,
		"hasTokenReviewerJwt": true,
		"gatewayId": null,
		"gatewayPoolId": "22222222-2222-2222-2222-222222222222",
		"allowedAudience": "infisical"
	}
}`

const oidcTemplateResponse = `{
	"id": "33333333-3333-3333-3333-333333333333",
	"name": "github-actions",
	"orgId": "99999999-9999-9999-9999-999999999999",
	"authMethod": "oidc",
	"templateFields": {
		"oidcDiscoveryUrl": "https://token.actions.githubusercontent.com",
		"boundIssuer": "https://token.actions.githubusercontent.com",
		"boundAudiences": "a, b",
		"caCert": ""
	}
}`

func TestCreateIdentityKubernetesAuthTemplateSendsMethodAndOmitsUnsetFields(t *testing.T) {
	var captured capturedRequest
	client := identityAuthTemplateServer(t, capturingHandler(t, &captured, kubernetesTemplateResponse))

	poolID := "22222222-2222-2222-2222-222222222222"
	template, err := client.CreateIdentityKubernetesAuthTemplate(CreateIdentityKubernetesAuthTemplateRequest{
		Name: "prod-cluster",
		TemplateFields: CreateIdentityKubernetesAuthTemplateFields{
			TokenReviewMode: "gateway",
			GatewayPoolID:   &poolID,
			AllowedAudience: "infisical",
		},
	})
	if err != nil {
		t.Fatalf("expected the template to be created, got: %v", err)
	}

	if captured.Method != http.MethodPost || captured.Path != "/api/v1/identity-templates" {
		t.Errorf("expected POST /api/v1/identity-templates, got %s %s", captured.Method, captured.Path)
	}
	if captured.Body["authMethod"] != "kubernetes" || captured.Body["name"] != "prod-cluster" {
		t.Errorf("expected the kubernetes method and name in the body, got %v", captured.Body)
	}

	fields, _ := captured.Body["templateFields"].(map[string]any)
	// A null gatewayId would be read as a gateway being cleared, and "" caCert or tokenReviewerJwt
	// as a stored empty value, so each unset field must be absent rather than zero.
	for _, key := range []string{"kubernetesHost", "caCert", "verifyTlsCertificate", "tokenReviewerJwt", "gatewayId"} {
		if _, present := fields[key]; present {
			t.Errorf("expected unset field %s to be omitted, got %v", key, fields[key])
		}
	}
	if fields["gatewayPoolId"] != poolID || fields["tokenReviewMode"] != "gateway" || fields["allowedAudience"] != "infisical" {
		t.Errorf("expected the set fields to be sent, got %v", fields)
	}

	if template.ID != "11111111-1111-1111-1111-111111111111" || template.Name != "prod-cluster" {
		t.Errorf("expected the created template back, got %+v", template)
	}
	if !template.TemplateFields.HasTokenReviewerJwt || template.TemplateFields.GatewayPoolID == nil || *template.TemplateFields.GatewayPoolID != poolID {
		t.Errorf("expected the response fields to decode, got %+v", template.TemplateFields)
	}
	if template.TemplateFields.KubernetesHost != nil {
		t.Errorf("expected a null host to decode as nil, got %q", *template.TemplateFields.KubernetesHost)
	}
}

func TestGetIdentityOidcAuthTemplateDecodesFields(t *testing.T) {
	var captured capturedRequest
	client := identityAuthTemplateServer(t, capturingHandler(t, &captured, oidcTemplateResponse))

	template, err := client.GetIdentityOidcAuthTemplate("33333333-3333-3333-3333-333333333333")
	if err != nil {
		t.Fatalf("expected the template to resolve, got: %v", err)
	}
	if captured.Path != "/api/v1/identity-templates/33333333-3333-3333-3333-333333333333" {
		t.Errorf("expected the template path, got %s", captured.Path)
	}
	if template.TemplateFields.BoundAudiences != "a, b" || template.TemplateFields.OidcDiscoveryUrl != "https://token.actions.githubusercontent.com" {
		t.Errorf("expected the OIDC fields to decode, got %+v", template.TemplateFields)
	}
}

// A template ID that belongs to another method must not be adopted as this one, since its fields
// would decode into the wrong shape without complaint.
func TestGetIdentityAuthTemplateRejectsAnotherMethod(t *testing.T) {
	client := identityAuthTemplateServer(t, jsonResponse(http.StatusOK, oidcTemplateResponse))

	_, err := client.GetIdentityKubernetesAuthTemplate("33333333-3333-3333-3333-333333333333")
	var mismatch *IdentityAuthTemplateMethodMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatalf("expected a method mismatch error, got: %v", err)
	}
	if mismatch.ActualMethod != "oidc" || mismatch.ExpectedMethod != "kubernetes" {
		t.Errorf("expected oidc reported against kubernetes, got %+v", mismatch)
	}
	if errors.Is(err, ErrNotFound) {
		t.Error("a template of another method exists, so it must not read as not found")
	}
}

func TestGetIdentityAuthTemplateNotFound(t *testing.T) {
	client := identityAuthTemplateServer(t, jsonResponse(http.StatusNotFound, `{"message":"Template not found"}`))

	if _, err := client.GetIdentityKubernetesAuthTemplate("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for a 404, got: %v", err)
	}
}

// Everything but a 404 must keep its cause, so a 403 is not mistaken for a deleted template and
// dropped from state.
func TestGetIdentityAuthTemplateFailureIsNotAbsence(t *testing.T) {
	client := identityAuthTemplateServer(t, jsonResponse(http.StatusForbidden, `{"message":"forbidden"}`))

	_, err := client.GetIdentityOidcAuthTemplate("33333333-3333-3333-3333-333333333333")
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("expected a non-ErrNotFound error for a 403, got: %v", err)
	}
	causes := collectAPIErrors(err)
	if len(causes) != 1 || causes[0].StatusCode != http.StatusForbidden {
		t.Errorf("expected the 403 to stay classifiable, got %v", causes)
	}
}

// The update is a merge, so a key left out keeps the stored value and a key set to nil clears it.
// Both have to survive serialization for the resource's patch to mean what it says.
func TestUpdateIdentityAuthTemplateSendsOnlyThePatch(t *testing.T) {
	var captured capturedRequest
	client := identityAuthTemplateServer(t, capturingHandler(t, &captured, kubernetesTemplateResponse))

	var cleared *string
	_, err := client.UpdateIdentityKubernetesAuthTemplate(UpdateIdentityAuthTemplateRequest{
		ID: "11111111-1111-1111-1111-111111111111",
		TemplateFields: map[string]any{
			"kubernetesHost":   cleared,
			"tokenReviewerJwt": "",
		},
	})
	if err != nil {
		t.Fatalf("expected the update to succeed, got: %v", err)
	}

	if captured.Method != http.MethodPatch || captured.Path != "/api/v1/identity-templates/11111111-1111-1111-1111-111111111111" {
		t.Errorf("expected PATCH on the template path, got %s %s", captured.Method, captured.Path)
	}
	if _, present := captured.Body["name"]; present {
		t.Errorf("expected an unchanged name to be omitted, got %v", captured.Body["name"])
	}
	fields, _ := captured.Body["templateFields"].(map[string]any)
	if len(fields) != 2 {
		t.Errorf("expected exactly the two patched keys, got %v", fields)
	}
	if value, present := fields["kubernetesHost"]; !present || value != nil {
		t.Errorf("expected kubernetesHost to be sent as null, got %v (present: %v)", value, present)
	}
	if fields["tokenReviewerJwt"] != "" {
		t.Errorf("expected tokenReviewerJwt to be sent as an empty string, got %v", fields["tokenReviewerJwt"])
	}
}

func TestUpdateIdentityAuthTemplateWithoutFieldChangesOmitsFields(t *testing.T) {
	var captured capturedRequest
	client := identityAuthTemplateServer(t, capturingHandler(t, &captured, oidcTemplateResponse))

	name := "renamed"
	if _, err := client.UpdateIdentityOidcAuthTemplate(UpdateIdentityAuthTemplateRequest{
		ID:             "33333333-3333-3333-3333-333333333333",
		Name:           &name,
		TemplateFields: map[string]any{},
	}); err != nil {
		t.Fatalf("expected the update to succeed, got: %v", err)
	}

	// An empty templateFields is harmless to the API, but leaving it out keeps a rename from
	// looking like a field edit that fans out to every linked identity.
	if _, present := captured.Body["templateFields"]; present {
		t.Errorf("expected an empty patch to be omitted, got %v", captured.Body["templateFields"])
	}
	if captured.Body["name"] != "renamed" {
		t.Errorf("expected the new name, got %v", captured.Body["name"])
	}
}

func TestFindIdentityAuthTemplateByName(t *testing.T) {
	list := `[
		{"id":"a","name":"prod","authMethod":"kubernetes","templateFields":{"tokenReviewMode":"api","hasTokenReviewerJwt":false,"allowedAudience":""}},
		{"id":"b","name":"Prod","authMethod":"kubernetes","templateFields":{"tokenReviewMode":"api","hasTokenReviewerJwt":false,"allowedAudience":""}},
		{"id":"c","name":"shared","authMethod":"kubernetes","templateFields":{"tokenReviewMode":"api","hasTokenReviewerJwt":false,"allowedAudience":""}},
		{"id":"d","name":"shared","authMethod":"kubernetes","templateFields":{"tokenReviewMode":"api","hasTokenReviewerJwt":false,"allowedAudience":""}}
	]`

	var captured capturedRequest
	client := identityAuthTemplateServer(t, capturingHandler(t, &captured, list))

	t.Run("exact match", func(t *testing.T) {
		template, err := client.GetIdentityKubernetesAuthTemplateByName("prod")
		if err != nil {
			t.Fatalf("expected the template to resolve, got: %v", err)
		}
		if template.ID != "a" {
			t.Errorf("expected the exactly matching template, got %s", template.ID)
		}
		if captured.Path != "/api/v1/identity-templates" || captured.Query != "authMethod=kubernetes" {
			t.Errorf("expected the list filtered by method, got %s?%s", captured.Path, captured.Query)
		}
	})

	t.Run("no match", func(t *testing.T) {
		if _, err := client.GetIdentityKubernetesAuthTemplateByName("PROD"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("expected ErrNotFound for a case-mismatched name, got: %v", err)
		}
	})

	// The API does not enforce unique names, so picking one would silently link the wrong template.
	t.Run("ambiguous", func(t *testing.T) {
		_, err := client.GetIdentityKubernetesAuthTemplateByName("shared")
		var ambiguous *IdentityAuthTemplateAmbiguousNameError
		if !errors.As(err, &ambiguous) {
			t.Fatalf("expected an ambiguous name error, got: %v", err)
		}
		if len(ambiguous.TemplateIDs) != 2 {
			t.Errorf("expected both matching IDs, got %v", ambiguous.TemplateIDs)
		}
	})
}

func TestFindIdentityAuthTemplateByNameListFailureIsNotAbsence(t *testing.T) {
	client := identityAuthTemplateServer(t, jsonResponse(http.StatusInternalServerError, `{"message":"boom"}`))

	_, err := client.GetIdentityOidcAuthTemplateByName("github-actions")
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("expected a list failure not to read as ErrNotFound, got: %v", err)
	}
}

func TestDeleteIdentityAuthTemplate(t *testing.T) {
	t.Run("deleted", func(t *testing.T) {
		var captured capturedRequest
		client := identityAuthTemplateServer(t, capturingHandler(t, &captured, `{"message":"Template deleted successfully"}`))

		if err := client.DeleteIdentityAuthTemplate("11111111-1111-1111-1111-111111111111"); err != nil {
			t.Fatalf("expected the delete to succeed, got: %v", err)
		}
		if captured.Method != http.MethodDelete || captured.Path != "/api/v1/identity-templates/11111111-1111-1111-1111-111111111111" {
			t.Errorf("expected DELETE on the template path, got %s %s", captured.Method, captured.Path)
		}
	})

	t.Run("already gone", func(t *testing.T) {
		client := identityAuthTemplateServer(t, jsonResponse(http.StatusNotFound, `{"message":"Template not found"}`))

		if err := client.DeleteIdentityAuthTemplate("missing"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("expected ErrNotFound for a 404, got: %v", err)
		}
	})
}

// A templated attach must not carry the template-managed fields at all: the API rejects them
// even as null.
func TestKubernetesAuthFromTemplateRequestCarriesNoManagedFields(t *testing.T) {
	var captured capturedRequest
	client := identityAuthTemplateServer(t, capturingHandler(t, &captured, `{"identityKubernetesAuth":{"id":"x","templateId":"t"}}`))

	auth, err := client.CreateIdentityKubernetesAuthFromTemplate(CreateIdentityKubernetesAuthFromTemplateRequest{
		IdentityID:        "identity",
		TemplateID:        "t",
		AllowedNamespaces: "default",
	})
	if err != nil {
		t.Fatalf("expected the attach to succeed, got: %v", err)
	}
	if captured.Path != "/api/v1/auth/kubernetes-auth/identities/identity" {
		t.Errorf("expected the identity path, got %s", captured.Path)
	}
	for _, key := range []string{"kubernetesHost", "caCert", "verifyTlsCertificate", "tokenReviewerJwt", "tokenReviewMode", "allowedAudience", "gatewayId", "gatewayPoolId"} {
		if _, present := captured.Body[key]; present {
			t.Errorf("expected template-managed field %s to be absent, got %v", key, captured.Body[key])
		}
	}
	if captured.Body["templateId"] != "t" {
		t.Errorf("expected the template ID, got %v", captured.Body["templateId"])
	}
	if auth.TemplateID == nil || *auth.TemplateID != "t" {
		t.Errorf("expected the linked template ID back, got %v", auth.TemplateID)
	}
}

func TestOidcAuthFromTemplateRequestCarriesNoManagedFields(t *testing.T) {
	var captured capturedRequest
	client := identityAuthTemplateServer(t, capturingHandler(t, &captured, `{"identityOidcAuth":{"id":"x","templateId":"t"}}`))

	if _, err := client.UpdateIdentityOidcAuthFromTemplate(UpdateIdentityOidcAuthFromTemplateRequest{
		IdentityID:   "identity",
		TemplateID:   "t",
		BoundSubject: "repo:org/repo:ref:refs/heads/main",
	}); err != nil {
		t.Fatalf("expected the update to succeed, got: %v", err)
	}
	if captured.Method != http.MethodPatch {
		t.Errorf("expected PATCH, got %s", captured.Method)
	}
	for _, key := range []string{"oidcDiscoveryUrl", "boundIssuer", "boundAudiences", "caCert"} {
		if _, present := captured.Body[key]; present {
			t.Errorf("expected template-managed field %s to be absent, got %v", key, captured.Body[key])
		}
	}
}

// A custom update must send templateId as null, which is what unlinks a template this identity
// was linked to.
func TestCustomAuthUpdatesSendTemplateIdNull(t *testing.T) {
	var captured capturedRequest
	client := identityAuthTemplateServer(t, capturingHandler(t, &captured, `{"identityKubernetesAuth":{"id":"x"}}`))

	if _, err := client.UpdateIdentityKubernetesAuth(UpdateIdentityKubernetesAuthRequest{IdentityID: "identity", TokenReviewerMode: "api"}); err != nil {
		t.Fatalf("expected the update to succeed, got: %v", err)
	}
	if value, present := captured.Body["templateId"]; !present || value != nil {
		t.Errorf("expected templateId to be sent as null, got %v (present: %v)", value, present)
	}

	captured = capturedRequest{}
	client = identityAuthTemplateServer(t, capturingHandler(t, &captured, `{"identityOidcAuth":{"id":"x"}}`))
	if _, err := client.UpdateIdentityOidcAuth(UpdateIdentityOidcAuthRequest{IdentityID: "identity"}); err != nil {
		t.Fatalf("expected the update to succeed, got: %v", err)
	}
	if value, present := captured.Body["templateId"]; !present || value != nil {
		t.Errorf("expected templateId to be sent as null, got %v (present: %v)", value, present)
	}
}
