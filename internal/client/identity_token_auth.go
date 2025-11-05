package infisicalclient

import (
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationGetIdentityTokenAuth    = "CallGetIdentityTokenAuth"
	operationCreateIdentityTokenAuth = "CallCreateIdentityTokenAuth"
	operationUpdateIdentityTokenAuth = "CallUpdateIdentityTokenAuth"
	operationRevokeIdentityTokenAuth = "CallRevokeIdentityTokenAuth"
)

func (client Client) GetIdentityTokenAuth(request GetIdentityTokenAuthRequest) (IdentityTokenAuth, error) {
	var body GetIdentityTokenAuthResponse

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT)

	response, err := httpRequest.Get("api/v1/auth/token-auth/identities/" + request.IdentityID)

	if response != nil && (response.StatusCode() == http.StatusNotFound || response.StatusCode() == http.StatusBadRequest) {
		return IdentityTokenAuth{}, ErrNotFound
	}

	if err != nil {
		return IdentityTokenAuth{}, errors.NewGenericRequestError(operationGetIdentityTokenAuth, err)
	}

	if response.IsError() {
		return IdentityTokenAuth{}, errors.NewAPIErrorWithResponse(operationGetIdentityTokenAuth, response, nil)
	}

	return body.IdentityTokenAuth, nil
}

func (client Client) CreateIdentityTokenAuth(request CreateIdentityTokenAuthRequest) (IdentityTokenAuth, error) {
	var body CreateIdentityTokenAuthResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/auth/token-auth/identities/" + request.IdentityID)

	if err != nil {
		return IdentityTokenAuth{}, errors.NewGenericRequestError(operationCreateIdentityTokenAuth, err)
	}

	if response.IsError() {
		return IdentityTokenAuth{}, errors.NewAPIErrorWithResponse(operationCreateIdentityTokenAuth, response, nil)
	}

	return body.IdentityTokenAuth, nil
}

func (client Client) UpdateIdentityTokenAuth(request UpdateIdentityTokenAuthRequest) (IdentityTokenAuth, error) {
	var body UpdateIdentityTokenAuthResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch("api/v1/auth/token-auth/identities/" + request.IdentityID)

	if err != nil {
		return IdentityTokenAuth{}, errors.NewGenericRequestError(operationUpdateIdentityTokenAuth, err)
	}

	if response.IsError() {
		return IdentityTokenAuth{}, errors.NewAPIErrorWithResponse(operationUpdateIdentityTokenAuth, response, nil)
	}

	return body.IdentityTokenAuth, nil
}

func (client Client) RevokeIdentityTokenAuth(request RevokeIdentityTokenAuthRequest) (IdentityTokenAuth, error) {
	var body RevokeIdentityTokenAuthResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Delete("api/v1/auth/token-auth/identities/" + request.IdentityID)

	if err != nil {
		return IdentityTokenAuth{}, errors.NewGenericRequestError(operationRevokeIdentityTokenAuth, err)
	}

	if response.IsError() {
		return IdentityTokenAuth{}, errors.NewAPIErrorWithResponse(operationRevokeIdentityTokenAuth, response, nil)
	}

	return body.IdentityTokenAuth, nil
}
