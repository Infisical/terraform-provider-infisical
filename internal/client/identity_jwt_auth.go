package infisicalclient

import (
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationGetIdentityJwtAuth    = "CallGetIdentityJwtAuth"
	operationCreateIdentityJwtAuth = "CallCreateIdentityJwtAuth"
	operationUpdateIdentityJwtAuth = "CallUpdateIdentityJwtAuth"
	operationRevokeIdentityJwtAuth = "CallRevokeIdentityJwtAuth"
)

func (client Client) CreateIdentityJwtAuth(request IdentityJwtAuthRequest) (IdentityJwtAuth, error) {
	var body IdentityJwtAuthResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/auth/jwt-auth/identities/" + request.IdentityID)

	if err != nil {
		return IdentityJwtAuth{}, errors.NewGenericRequestError(operationCreateIdentityJwtAuth, err)
	}

	if response.IsError() {
		return IdentityJwtAuth{}, errors.NewAPIErrorWithResponse(operationCreateIdentityJwtAuth, response, nil)
	}

	return body.IdentityJwtAuth, nil
}

func (client Client) GetIdentityJwtAuth(request GetIdentityJwtAuthRequest) (IdentityJwtAuth, error) {
	var body IdentityJwtAuthResponse

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT)

	response, err := httpRequest.Get("api/v1/auth/jwt-auth/identities/" + request.IdentityID)

	if err != nil {
		return IdentityJwtAuth{}, errors.NewGenericRequestError(operationGetIdentityJwtAuth, err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return IdentityJwtAuth{}, ErrNotFound
	}

	if response.IsError() {
		return IdentityJwtAuth{}, errors.NewAPIErrorWithResponse(operationGetIdentityJwtAuth, response, nil)
	}

	return body.IdentityJwtAuth, nil
}

func (client Client) UpdateIdentityJwtAuth(request IdentityJwtAuthRequest) (IdentityJwtAuth, error) {
	var body IdentityJwtAuthResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch("api/v1/auth/jwt-auth/identities/" + request.IdentityID)

	if err != nil {
		return IdentityJwtAuth{}, errors.NewGenericRequestError(operationUpdateIdentityJwtAuth, err)
	}

	if response.IsError() {
		return IdentityJwtAuth{}, errors.NewAPIErrorWithResponse(operationUpdateIdentityJwtAuth, response, nil)
	}

	return body.IdentityJwtAuth, nil
}

func (client Client) RevokeIdentityJwtAuth(request RevokeIdentityJwtAuthRequest) (IdentityJwtAuth, error) {
	var body IdentityJwtAuthResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Delete("api/v1/auth/jwt-auth/identities/" + request.IdentityID)

	if err != nil {
		return IdentityJwtAuth{}, errors.NewGenericRequestError(operationRevokeIdentityJwtAuth, err)
	}

	if response.IsError() {
		return IdentityJwtAuth{}, errors.NewAPIErrorWithResponse(operationRevokeIdentityJwtAuth, response, nil)
	}

	return body.IdentityJwtAuth, nil
}
