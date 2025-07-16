package infisicalclient

import (
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationGetIdentityGcpAuth    = "CallGetIdentityGcpAuth"
	operationCreateIdentityGcpAuth = "CallCreateIdentityGcpAuth"
	operationUpdateIdentityGcpAuth = "CallUpdateIdentityGcpAuth"
	operationRevokeIdentityGcpAuth = "CallRevokeIdentityGcpAuth"
)

func (client Client) GetIdentityGcpAuth(request GetIdentityGcpAuthRequest) (IdentityGcpAuth, error) {
	var body GetIdentityGcpAuthResponse

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT)

	response, err := httpRequest.Get("api/v1/auth/gcp-auth/identities/" + request.IdentityID)

	if response.StatusCode() == http.StatusNotFound {
		return IdentityGcpAuth{}, ErrNotFound
	}

	if err != nil {
		return IdentityGcpAuth{}, errors.NewGenericRequestError(operationGetIdentityGcpAuth, err)
	}

	if response.IsError() {
		return IdentityGcpAuth{}, errors.NewAPIErrorWithResponse(operationGetIdentityGcpAuth, response, nil)
	}

	return body.IdentityGcpAuth, nil
}

func (client Client) CreateIdentityGcpAuth(request CreateIdentityGcpAuthRequest) (IdentityGcpAuth, error) {
	var body CreateIdentityGcpAuthResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/auth/gcp-auth/identities/" + request.IdentityID)

	if err != nil {
		return IdentityGcpAuth{}, errors.NewGenericRequestError(operationCreateIdentityGcpAuth, err)
	}

	if response.IsError() {
		return IdentityGcpAuth{}, errors.NewAPIErrorWithResponse(operationCreateIdentityGcpAuth, response, nil)
	}

	return body.IdentityGcpAuth, nil
}

func (client Client) UpdateIdentityGcpAuth(request UpdateIdentityGcpAuthRequest) (IdentityGcpAuth, error) {
	var body UpdateIdentityGcpAuthResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch("api/v1/auth/gcp-auth/identities/" + request.IdentityID)

	if err != nil {
		return IdentityGcpAuth{}, errors.NewGenericRequestError(operationUpdateIdentityGcpAuth, err)
	}

	if response.IsError() {
		return IdentityGcpAuth{}, errors.NewAPIErrorWithResponse(operationUpdateIdentityGcpAuth, response, nil)
	}

	return body.IdentityGcpAuth, nil
}

func (client Client) RevokeIdentityGcpAuth(request RevokeIdentityGcpAuthRequest) (IdentityGcpAuth, error) {
	var body RevokeIdentityGcpAuthResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Delete("api/v1/auth/gcp-auth/identities/" + request.IdentityID)

	if err != nil {
		return IdentityGcpAuth{}, errors.NewGenericRequestError(operationRevokeIdentityGcpAuth, err)
	}

	if response.IsError() {
		return IdentityGcpAuth{}, errors.NewAPIErrorWithResponse(operationRevokeIdentityGcpAuth, response, nil)
	}

	return body.IdentityGcpAuth, nil
}
