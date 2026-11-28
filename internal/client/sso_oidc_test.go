package infisicalclient

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"testing"

	infisicalerrors "terraform-provider-infisical/internal/errors"
)

type requestRecorder struct {
	method, path string
	body         map[string]any
}

func (rec *requestRecorder) handler(t *testing.T, response string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rec.method, rec.path, rec.body = r.Method, r.URL.Path, nil
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &rec.body); err != nil {
				t.Error(err)
			}
		}
		jsonResponse(http.StatusOK, response)(w, r)
	}
}

func TestGetOidcConfig(t *testing.T) {
	rec := &requestRecorder{}
	client := testClient(t, rec.handler(t, `{"id":"c1","orgId":"o1","configurationType":"discoveryURL","discoveryURL":"https://idp.example.com","issuer":null,"clientId":"id","clientSecret":"secret","allowedEmailDomains":"example.com, example.org","jwtSignatureAlgorithm":"RS256","isActive":true,"mfaClaimName":"amr","mfaAmrValues":null}`))

	config, err := client.GetOidcConfig()
	if err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodGet || rec.path != "/api/v1/sso/oidc/config" {
		t.Errorf("unexpected request: %s %s", rec.method, rec.path)
	}
	if config.ID != "c1" || config.Issuer != "" || config.ClientSecret != "secret" || config.MfaAmrValues != nil || !config.IsActive {
		t.Errorf("unexpected config: %+v", config)
	}

	notFound := testClient(t, jsonResponse(http.StatusNotFound, `{"statusCode":404,"message":"Failed to find OIDC SSO data"}`))
	if _, err := notFound.GetOidcConfig(); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestCreateOidcConfigRequest(t *testing.T) {
	rec := &requestRecorder{}
	client := testClient(t, rec.handler(t, `{"id":"c1","orgId":"o1"}`))

	_, err := client.CreateOidcConfig(CreateOidcConfigRequest{
		ConfigurationType:     OidcConfigurationTypeDiscoveryURL,
		DiscoveryURL:          "https://idp.example.com",
		ClientID:              "id",
		ClientSecret:          "secret",
		JwtSignatureAlgorithm: "RS256",
		MfaClaimName:          "amr",
	})
	if err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodPost || rec.path != "/api/v1/sso/oidc/config" {
		t.Errorf("unexpected request: %s %s", rec.method, rec.path)
	}
	want := map[string]any{
		"configurationType":      "discoveryURL",
		"discoveryURL":           "https://idp.example.com",
		"clientId":               "id",
		"clientSecret":           "secret",
		"allowedEmailDomains":    "",
		"jwtSignatureAlgorithm":  "RS256",
		"manageGroupMemberships": false,
		"isActive":               false,
		"mfaClaimName":           "amr",
	}
	if !reflect.DeepEqual(rec.body, want) {
		t.Errorf("unexpected request body:\n got: %v\nwant: %v", rec.body, want)
	}

	conflict := testClient(t, jsonResponse(http.StatusConflict, `{"statusCode":409,"message":"An OIDC configuration already exists."}`))
	_, err = conflict.CreateOidcConfig(CreateOidcConfigRequest{})
	var apiErr *infisicalerrors.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict {
		t.Errorf("expected a 409 APIError, got: %v", err)
	}
}

func TestUpdateOidcConfigSendsFullConfig(t *testing.T) {
	rec := &requestRecorder{}
	client := testClient(t, rec.handler(t, `{"id":"c1","jwtSignatureAlgorithm":"RS512"}`))

	config, err := client.UpdateOidcConfig(UpdateOidcConfigRequest{
		ConfigurationType:     OidcConfigurationTypeDiscoveryURL,
		DiscoveryURL:          "https://idp.example.com/.well-known/openid-configuration",
		ClientID:              "client-id",
		ClientSecret:          "client-secret",
		JwtSignatureAlgorithm: "RS512",
		MfaClaimName:          "amr",
		MfaAmrValues:          []string{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if config.ID != "c1" || config.JwtSignatureAlgorithm != "RS512" {
		t.Errorf("unexpected response: %+v", config)
	}
	if rec.method != http.MethodPatch || rec.path != "/api/v1/sso/oidc/config" {
		t.Errorf("unexpected request: %s %s", rec.method, rec.path)
	}
	want := map[string]any{
		"configurationType":      "discoveryURL",
		"discoveryURL":           "https://idp.example.com/.well-known/openid-configuration",
		"clientId":               "client-id",
		"clientSecret":           "client-secret",
		"allowedEmailDomains":    "",
		"jwtSignatureAlgorithm":  "RS512",
		"manageGroupMemberships": false,
		"isActive":               false,
		"mfaClaimName":           "amr",
		"mfaAmrValues":           []any{},
	}
	if !reflect.DeepEqual(rec.body, want) {
		t.Errorf("unexpected request body:\n got: %v\nwant: %v", rec.body, want)
	}

	notFound := testClient(t, jsonResponse(http.StatusNotFound, `{"statusCode":404,"message":"OIDC configuration not found"}`))
	if _, err := notFound.UpdateOidcConfig(UpdateOidcConfigRequest{}); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestDeleteOidcConfig(t *testing.T) {
	rec := &requestRecorder{}
	client := testClient(t, rec.handler(t, `{"id":"c1"}`))

	if err := client.DeleteOidcConfig(); err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodDelete || rec.path != "/api/v1/sso/oidc/config" {
		t.Errorf("unexpected request: %s %s", rec.method, rec.path)
	}

	notFound := testClient(t, jsonResponse(http.StatusNotFound, `{"statusCode":404,"message":"OIDC configuration not found"}`))
	if err := notFound.DeleteOidcConfig(); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}
