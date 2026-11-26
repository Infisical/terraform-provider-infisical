package infisicalclient

import (
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationGetOrgSsoSettings    = "CallGetOrgSsoSettings"
	operationUpdateOrgSsoSettings = "CallUpdateOrgSsoSettings"
)

type SsoMfaTrust string

const (
	SsoMfaTrustNone   SsoMfaTrust = "none"
	SsoMfaTrustSignal SsoMfaTrust = "signal"
)

func (client Client) GetOrgSsoSettings() (OrgSsoSettings, error) {
	var body GetOrgSsoSettingsResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Get("api/v1/organization/sso-settings")

	if err != nil {
		return OrgSsoSettings{}, errors.NewGenericRequestError(operationGetOrgSsoSettings, err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return OrgSsoSettings{}, ErrNotFound
	}

	if response.IsError() {
		return OrgSsoSettings{}, errors.NewAPIErrorWithResponse(operationGetOrgSsoSettings, response, nil)
	}

	return body.SsoSettings, nil
}

func (client Client) UpdateOrgSsoSettings(request UpdateOrgSsoSettingsRequest) (OrgSsoSettings, error) {
	var body UpdateOrgSsoSettingsResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch("api/v1/organization/sso-settings")

	if err != nil {
		return OrgSsoSettings{}, errors.NewGenericRequestError(operationUpdateOrgSsoSettings, err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return OrgSsoSettings{}, ErrNotFound
	}

	if response.IsError() {
		return OrgSsoSettings{}, errors.NewAPIErrorWithResponse(operationUpdateOrgSsoSettings, response, nil)
	}

	return body.SsoSettings, nil
}
