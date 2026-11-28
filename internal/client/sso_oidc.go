package infisicalclient

import (
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationGetOidcConfig    = "CallGetOidcConfig"
	operationCreateOidcConfig = "CallCreateOidcConfig"
	operationUpdateOidcConfig = "CallUpdateOidcConfig"
	operationDeleteOidcConfig = "CallDeleteOidcConfig"
)

const (
	OidcConfigurationTypeDiscoveryURL = "discoveryURL"
	OidcConfigurationTypeCustom       = "custom"
)

func (client Client) GetOidcConfig() (OidcConfig, error) {
	var responseData OidcConfig
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		Get("api/v1/sso/oidc/config")

	if err != nil {
		return OidcConfig{}, errors.NewGenericRequestError(operationGetOidcConfig, err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return OidcConfig{}, ErrNotFound
	}

	if response.IsError() {
		return OidcConfig{}, errors.NewAPIErrorWithResponse(operationGetOidcConfig, response, nil)
	}

	return responseData, nil
}

func (client Client) CreateOidcConfig(request CreateOidcConfigRequest) (OidcConfig, error) {
	var responseData OidcConfig
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/sso/oidc/config")

	if err != nil {
		return OidcConfig{}, errors.NewGenericRequestError(operationCreateOidcConfig, err)
	}

	if response.IsError() {
		return OidcConfig{}, errors.NewAPIErrorWithResponse(operationCreateOidcConfig, response, nil)
	}

	return responseData, nil
}

func (client Client) UpdateOidcConfig(request UpdateOidcConfigRequest) (OidcConfig, error) {
	var responseData OidcConfig
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch("api/v1/sso/oidc/config")

	if err != nil {
		return OidcConfig{}, errors.NewGenericRequestError(operationUpdateOidcConfig, err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return OidcConfig{}, ErrNotFound
	}

	if response.IsError() {
		return OidcConfig{}, errors.NewAPIErrorWithResponse(operationUpdateOidcConfig, response, nil)
	}

	return responseData, nil
}

func (client Client) DeleteOidcConfig() error {
	response, err := client.Config.HttpClient.
		R().
		SetHeader("User-Agent", USER_AGENT).
		Delete("api/v1/sso/oidc/config")

	if err != nil {
		return errors.NewGenericRequestError(operationDeleteOidcConfig, err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return ErrNotFound
	}

	if response.IsError() {
		return errors.NewAPIErrorWithResponse(operationDeleteOidcConfig, response, nil)
	}

	return nil
}
