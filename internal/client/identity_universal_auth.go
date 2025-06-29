package infisicalclient

import (
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationGetIdentityUniversalAuth    = "CallGetIdentityUniversalAuth"
	operationCreateIdentityUniversalAuth = "CallCreateIdentityUniversalAuth"
	operationUpdateIdentityUniversalAuth = "CallUpdateIdentityUniversalAuth"
	operationRevokeIdentityUniversalAuth = "CallRevokeIdentityUniversalAuth"
)

func (client Client) GetIdentityUniversalAuth(request GetIdentityUniversalAuthRequest) (IdentityUniversalAuth, error) {
	var body GetIdentityUniversalAuthResponse

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT)

	response, err := httpRequest.Get("api/v1/auth/universal-auth/identities/" + request.IdentityID)

	if response.StatusCode() == http.StatusNotFound {
		return IdentityUniversalAuth{}, ErrNotFound
	}

	if err != nil {
		return IdentityUniversalAuth{}, errors.NewGenericRequestError(operationGetIdentityUniversalAuth, err)
	}

	if response.IsError() {
		return IdentityUniversalAuth{}, errors.NewAPIErrorWithResponse(operationGetIdentityUniversalAuth, response, nil)
	}

	return body.UniversalAuth, nil
}

func (client Client) CreateIdentityUniversalAuth(request CreateIdentityUniversalAuthRequest) (IdentityUniversalAuth, error) {
	var body CreateIdentityUniversalAuthResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/auth/universal-auth/identities/" + request.IdentityID)

	if err != nil {
		return IdentityUniversalAuth{}, errors.NewGenericRequestError(operationCreateIdentityUniversalAuth, err)
	}

	if response.IsError() {
		return IdentityUniversalAuth{}, errors.NewAPIErrorWithResponse(operationCreateIdentityUniversalAuth, response, nil)
	}

	return body.UniversalAuth, nil
}

func (client Client) UpdateIdentityUniversalAuth(request UpdateIdentityUniversalAuthRequest) (IdentityUniversalAuth, error) {
	var body UpdateIdentityUniversalAuthResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch("api/v1/auth/universal-auth/identities/" + request.IdentityID)

	if err != nil {
		return IdentityUniversalAuth{}, errors.NewGenericRequestError(operationUpdateIdentityUniversalAuth, err)
	}

	if response.IsError() {
		return IdentityUniversalAuth{}, errors.NewAPIErrorWithResponse(operationUpdateIdentityUniversalAuth, response, nil)
	}

	return body.UniversalAuth, nil
}

func (client Client) RevokeIdentityUniversalAuth(request RevokeIdentityUniversalAuthRequest) (IdentityUniversalAuth, error) {
	var body RevokeIdentityUniversalAuthResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Delete("api/v1/auth/universal-auth/identities/" + request.IdentityID)

	if err != nil {
		return IdentityUniversalAuth{}, errors.NewGenericRequestError(operationRevokeIdentityUniversalAuth, err)
	}

	if response.IsError() {
		return IdentityUniversalAuth{}, errors.NewAPIErrorWithResponse(operationRevokeIdentityUniversalAuth, response, nil)
	}

	return body.UniversalAuth, nil
}
