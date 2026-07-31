package infisicalclient

import (
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationGetIdentityTlsCertAuth    = "CallGetIdentityTlsCertAuth"
	operationCreateIdentityTlsCertAuth = "CallCreateIdentityTlsCertAuth"
	operationUpdateIdentityTlsCertAuth = "CallUpdateIdentityTlsCertAuth"
	operationRevokeIdentityTlsCertAuth = "CallRevokeIdentityTlsCertAuth"
)

func (client Client) GetIdentityTlsCertAuth(request GetIdentityTlsCertAuthRequest) (IdentityTlsCertAuth, error) {
	var body GetIdentityTlsCertAuthResponse

	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Get("api/v1/auth/tls-cert-auth/identities/" + request.IdentityID)

	if err != nil {
		return IdentityTlsCertAuth{}, errors.NewGenericRequestError(operationGetIdentityTlsCertAuth, err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return IdentityTlsCertAuth{}, ErrNotFound
	}

	if response.IsError() {
		return IdentityTlsCertAuth{}, errors.NewAPIErrorWithResponse(operationGetIdentityTlsCertAuth, response, nil)
	}

	return body.IdentityTlsCertAuth, nil
}

func (client Client) CreateIdentityTlsCertAuth(request CreateIdentityTlsCertAuthRequest) (IdentityTlsCertAuth, error) {
	var body CreateIdentityTlsCertAuthResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/auth/tls-cert-auth/identities/" + request.IdentityID)

	if err != nil {
		return IdentityTlsCertAuth{}, errors.NewGenericRequestError(operationCreateIdentityTlsCertAuth, err)
	}

	if response.IsError() {
		return IdentityTlsCertAuth{}, errors.NewAPIErrorWithResponse(operationCreateIdentityTlsCertAuth, response, nil)
	}

	return body.IdentityTlsCertAuth, nil
}

func (client Client) UpdateIdentityTlsCertAuth(request UpdateIdentityTlsCertAuthRequest) (IdentityTlsCertAuth, error) {
	var body UpdateIdentityTlsCertAuthResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch("api/v1/auth/tls-cert-auth/identities/" + request.IdentityID)

	if err != nil {
		return IdentityTlsCertAuth{}, errors.NewGenericRequestError(operationUpdateIdentityTlsCertAuth, err)
	}

	if response.IsError() {
		return IdentityTlsCertAuth{}, errors.NewAPIErrorWithResponse(operationUpdateIdentityTlsCertAuth, response, nil)
	}

	return body.IdentityTlsCertAuth, nil
}

func (client Client) RevokeIdentityTlsCertAuth(request RevokeIdentityTlsCertAuthRequest) (IdentityTlsCertAuth, error) {
	var body RevokeIdentityTlsCertAuthResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Delete("api/v1/auth/tls-cert-auth/identities/" + request.IdentityID)

	if err != nil {
		return IdentityTlsCertAuth{}, errors.NewGenericRequestError(operationRevokeIdentityTlsCertAuth, err)
	}

	if response.IsError() {
		return IdentityTlsCertAuth{}, errors.NewAPIErrorWithResponse(operationRevokeIdentityTlsCertAuth, response, nil)
	}

	return body.IdentityTlsCertAuth, nil
}
