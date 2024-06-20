package infisicalclient

import (
	"fmt"
	"net/http"
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
		return IdentityAzureAuth{}, fmt.Errorf("GetIdentityAzureAuth: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return IdentityAzureAuth{}, fmt.Errorf("GetIdentityAzureAuth: Unsuccessful response. [response=%v]", string(response.Body()))
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
		return IdentityAzureAuth{}, fmt.Errorf("CreateIdentityAzureAuth: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return IdentityAzureAuth{}, fmt.Errorf("CreateIdentityAzureAuth: Unsuccessful response. [response=%s]", string(response.Body()))
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
		return IdentityAzureAuth{}, fmt.Errorf("UpdateIdentityAzureAuth: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return IdentityAzureAuth{}, fmt.Errorf("UpdateIdentityAzureAuth: Unsuccessful response. [response=%s]", string(response.Body()))
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
		return IdentityAzureAuth{}, fmt.Errorf("RevokeIdentityAzureAuth: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return IdentityAzureAuth{}, fmt.Errorf("RevokeIdentityAzureAuth: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body.IdentityAzureAuth, nil
}
