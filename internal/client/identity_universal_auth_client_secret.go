package infisicalclient

import (
	"fmt"
	"net/http"
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
		return IdentityUniversalAuthClientSecret{}, fmt.Errorf("GetIdentityUniversalAuthClientSecret: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return IdentityUniversalAuthClientSecret{}, fmt.Errorf("GetIdentityUniversalAuthClientSecret: Unsuccessful response. [response=%v]", string(response.Body()))
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
		return CreateIdentityUniversalAuthClientSecretResponse{}, fmt.Errorf("CreateIdentityUniversalAuthClientSecret: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return CreateIdentityUniversalAuthClientSecretResponse{}, fmt.Errorf("CreateIdentityUniversalAuthClientSecret: Unsuccessful response. [response=%s]", string(response.Body()))
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
		return IdentityUniversalAuthClientSecret{}, fmt.Errorf("RevokeIdentityUniversalAuthClientSecret: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return IdentityUniversalAuthClientSecret{}, fmt.Errorf("RevokeIdentityUniversalAuthClientSecret: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body.ClientSecretData, nil
}
