package infisicalclient

import (
	"errors"
	"net/http"
	"reflect"
	"testing"

	infisicalerrors "terraform-provider-infisical/internal/errors"
)

const ssoSettingsJSON = `{"ssoSettings":{"orgId":"o1","authEnforced":true,"googleSsoAuthEnforced":false,"bypassOrgAuthEnabled":true,"ssoMfaTrust":"signal"}}`

func TestGetOrgSsoSettings(t *testing.T) {
	rec := &requestRecorder{}
	client := testClient(t, rec.handler(t, ssoSettingsJSON))

	settings, err := client.GetOrgSsoSettings()
	if err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodGet || rec.path != "/api/v1/organization/sso-settings" {
		t.Errorf("unexpected request: %s %s", rec.method, rec.path)
	}
	want := OrgSsoSettings{OrgID: "o1", AuthEnforced: true, BypassOrgAuthEnabled: true, SsoMfaTrust: "signal"}
	if settings != want {
		t.Errorf("unexpected settings: %+v", settings)
	}

	notFound := testClient(t, jsonResponse(http.StatusNotFound, `{"statusCode":404,"message":"Organization with ID 'o1' not found"}`))
	if _, err := notFound.GetOrgSsoSettings(); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

// The API leaves an omitted field unchanged, so false and "none" must be in the body.
func TestUpdateOrgSsoSettingsSendsZeroValues(t *testing.T) {
	rec := &requestRecorder{}
	client := testClient(t, rec.handler(t, ssoSettingsJSON))

	if _, err := client.UpdateOrgSsoSettings(UpdateOrgSsoSettingsRequest{SsoMfaTrust: SsoMfaTrustNone}); err != nil {
		t.Fatal(err)
	}
	if rec.method != http.MethodPatch || rec.path != "/api/v1/organization/sso-settings" {
		t.Errorf("unexpected request: %s %s", rec.method, rec.path)
	}
	want := map[string]any{
		"authEnforced":          false,
		"googleSsoAuthEnforced": false,
		"bypassOrgAuthEnabled":  false,
		"ssoMfaTrust":           "none",
	}
	if !reflect.DeepEqual(rec.body, want) {
		t.Errorf("unexpected body:\n got: %v\nwant: %v", rec.body, want)
	}
}

func TestUpdateOrgSsoSettingsErrors(t *testing.T) {
	badRequest := testClient(t, jsonResponse(http.StatusBadRequest, `{"statusCode":400,"message":"To apply the new OIDC auth enforcement, please log in via OIDC at least once."}`))
	_, err := badRequest.UpdateOrgSsoSettings(UpdateOrgSsoSettingsRequest{AuthEnforced: true, SsoMfaTrust: SsoMfaTrustNone})
	var apiErr *infisicalerrors.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("expected a 400 APIError, got: %v", err)
	}

	notFound := testClient(t, jsonResponse(http.StatusNotFound, `{"statusCode":404,"message":"Organization with ID 'o1' not found"}`))
	if _, err := notFound.UpdateOrgSsoSettings(UpdateOrgSsoSettingsRequest{SsoMfaTrust: SsoMfaTrustNone}); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}
