package infisicalclient

import (
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationGetIdentityUniversalAuthClientSecret    = "CallGetIdentityUniversalAuthClientSecret"
	operationCreateIdentityUniversalAuthClientSecret = "CallCreateIdentityUniversalAuthClientSecret"
	operationRevokeIdentityUniversalAuthClientSecret = "CallRevokeIdentityUniversalAuthClientSecret"
)

func (client Client) GetIdentityUniversalAuthClientSecret(request GetIdentityUniversalAuthClientSecretRequest) (IdentityUniversalAuthClientSecret, error) {
	var body GetIdentityUniversalAuthClientSecretResponse

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT)

	response, err := httpRequest.Get("api/v1/auth/universal-auth/identities/" + request.IdentityID + "/client-secrets/" + request.ClientSecretID)

	if response.StatusCode() == http.StatusNotFound {
		return IdentityUniversalAuthClientSecret{}, ErrNotFound
	}

	if err != nil {
		return IdentityUniversalAuthClientSecret{}, errors.NewGenericRequestError(operationGetIdentityUniversalAuthClientSecret, err)
	}

	if response.IsError() {
		return IdentityUniversalAuthClientSecret{}, errors.NewAPIErrorWithResponse(operationGetIdentityUniversalAuthClientSecret, response, nil)
	}

	return body.ClientSecretData, nil
}

func (client Client) CreateIdentityUniversalAuthClientSecret(request CreateIdentityUniversalAuthClientSecretRequest) (CreateIdentityUniversalAuthClientSecretResponse, error) {
	var body CreateIdentityUniversalAuthClientSecretResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/auth/universal-auth/identities/" + request.IdentityID + "/client-secrets")

	if err != nil {
		return CreateIdentityUniversalAuthClientSecretResponse{}, errors.NewGenericRequestError(operationCreateIdentityUniversalAuthClientSecret, err)
	}

	if response.IsError() {
		return CreateIdentityUniversalAuthClientSecretResponse{}, errors.NewAPIErrorWithResponse(operationCreateIdentityUniversalAuthClientSecret, response, nil)
	}

	return body, nil
}

func (client Client) RevokeIdentityUniversalAuthClientSecret(request RevokeIdentityUniversalAuthClientSecretRequest) (IdentityUniversalAuthClientSecret, error) {
	var body RevokeIdentityUniversalAuthClientSecretResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/auth/universal-auth/identities/" + request.IdentityID + "/client-secrets/" + request.ClientSecretID + "/revoke")

	if err != nil {
		return IdentityUniversalAuthClientSecret{}, errors.NewGenericRequestError(operationRevokeIdentityUniversalAuthClientSecret, err)
	}

	if response.IsError() {
		return IdentityUniversalAuthClientSecret{}, errors.NewAPIErrorWithResponse(operationRevokeIdentityUniversalAuthClientSecret, response, nil)
	}

	return body.ClientSecretData, nil
}
