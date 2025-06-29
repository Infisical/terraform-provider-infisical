package infisicalclient

import (
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationGetIdentityAzureAuth    = "CallGetIdentityAzureAuth"
	operationCreateIdentityAzureAuth = "CallCreateIdentityAzureAuth"
	operationUpdateIdentityAzureAuth = "CallUpdateIdentityAzureAuth"
	operationRevokeIdentityAzureAuth = "CallRevokeIdentityAzureAuth"
)

func (client Client) GetIdentityAzureAuth(request GetIdentityAzureAuthRequest) (IdentityAzureAuth, error) {
	var body GetIdentityAzureAuthResponse

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT)

	response, err := httpRequest.Get("api/v1/auth/azure-auth/identities/" + request.IdentityID)

	if response.StatusCode() == http.StatusNotFound {
		return IdentityAzureAuth{}, ErrNotFound
	}

	if err != nil {
		return IdentityAzureAuth{}, errors.NewGenericRequestError(operationGetIdentityAzureAuth, err)
	}

	if response.IsError() {
		return IdentityAzureAuth{}, errors.NewAPIErrorWithResponse(operationGetIdentityAzureAuth, response, nil)
	}

	return body.IdentityAzureAuth, nil
}

func (client Client) CreateIdentityAzureAuth(request CreateIdentityAzureAuthRequest) (IdentityAzureAuth, error) {
	var body CreateIdentityAzureAuthResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/auth/azure-auth/identities/" + request.IdentityID)

	if err != nil {
		return IdentityAzureAuth{}, errors.NewGenericRequestError(operationCreateIdentityAzureAuth, err)
	}

	if response.IsError() {
		return IdentityAzureAuth{}, errors.NewAPIErrorWithResponse(operationCreateIdentityAzureAuth, response, nil)
	}

	return body.IdentityAzureAuth, nil
}

func (client Client) UpdateIdentityAzureAuth(request UpdateIdentityAzureAuthRequest) (IdentityAzureAuth, error) {
	var body UpdateIdentityAzureAuthResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch("api/v1/auth/azure-auth/identities/" + request.IdentityID)

	if err != nil {
		return IdentityAzureAuth{}, errors.NewGenericRequestError(operationUpdateIdentityAzureAuth, err)
	}

	if response.IsError() {
		return IdentityAzureAuth{}, errors.NewAPIErrorWithResponse(operationUpdateIdentityAzureAuth, response, nil)
	}

	return body.IdentityAzureAuth, nil
}

func (client Client) RevokeIdentityAzureAuth(request RevokeIdentityAzureAuthRequest) (IdentityAzureAuth, error) {
	var body RevokeIdentityAzureAuthResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Delete("api/v1/auth/azure-auth/identities/" + request.IdentityID)

	if err != nil {
		return IdentityAzureAuth{}, errors.NewGenericRequestError(operationRevokeIdentityAzureAuth, err)
	}

	if response.IsError() {
		return IdentityAzureAuth{}, errors.NewAPIErrorWithResponse(operationRevokeIdentityAzureAuth, response, nil)
	}

	return body.IdentityAzureAuth, nil
}
