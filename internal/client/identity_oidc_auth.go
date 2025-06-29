package infisicalclient

import (
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationGetIdentityOidcAuth    = "CallGetIdentityOidcAuth"
	operationCreateIdentityOidcAuth = "CallCreateIdentityOidcAuth"
	operationUpdateIdentityOidcAuth = "CallUpdateIdentityOidcAuth"
	operationRevokeIdentityOidcAuth = "CallRevokeIdentityOidcAuth"
)

func (client Client) CreateIdentityOidcAuth(request CreateIdentityOidcAuthRequest) (IdentityOidcAuth, error) {
	var body CreateIdentityOidcAuthResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/auth/oidc-auth/identities/" + request.IdentityID)

	if err != nil {
		return IdentityOidcAuth{}, errors.NewGenericRequestError(operationCreateIdentityOidcAuth, err)
	}

	if response.IsError() {
		return IdentityOidcAuth{}, errors.NewAPIErrorWithResponse(operationCreateIdentityOidcAuth, response, nil)
	}

	return body.IdentityOidcAuth, nil
}

func (client Client) GetIdentityOidcAuth(request GetIdentityOidcAuthRequest) (IdentityOidcAuth, error) {
	var body GetIdentityOidcAuthResponse

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT)

	response, err := httpRequest.Get("api/v1/auth/oidc-auth/identities/" + request.IdentityID)

	if response.StatusCode() == http.StatusNotFound {
		return IdentityOidcAuth{}, ErrNotFound
	}

	if err != nil {
		return IdentityOidcAuth{}, errors.NewGenericRequestError(operationGetIdentityOidcAuth, err)
	}

	if response.IsError() {
		return IdentityOidcAuth{}, errors.NewAPIErrorWithResponse(operationGetIdentityOidcAuth, response, nil)
	}

	return body.IdentityOidcAuth, nil
}

func (client Client) UpdateIdentityOidcAuth(request UpdateIdentityOidcAuthRequest) (IdentityOidcAuth, error) {
	var body UpdateIdentityOidcAuthResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch("api/v1/auth/oidc-auth/identities/" + request.IdentityID)

	if err != nil {
		return IdentityOidcAuth{}, errors.NewGenericRequestError(operationUpdateIdentityOidcAuth, err)
	}

	if response.IsError() {
		return IdentityOidcAuth{}, errors.NewAPIErrorWithResponse(operationUpdateIdentityOidcAuth, response, nil)
	}

	return body.IdentityOidcAuth, nil
}

func (client Client) RevokeIdentityOidcAuth(request RevokeIdentityOidcAuthRequest) (IdentityOidcAuth, error) {
	var body RevokeIdentityOidcAuthResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Delete("api/v1/auth/oidc-auth/identities/" + request.IdentityID)

	if err != nil {
		return IdentityOidcAuth{}, errors.NewGenericRequestError(operationRevokeIdentityOidcAuth, err)
	}

	if response.IsError() {
		return IdentityOidcAuth{}, errors.NewAPIErrorWithResponse(operationRevokeIdentityOidcAuth, response, nil)
	}

	return body.IdentityOidcAuth, nil
}
