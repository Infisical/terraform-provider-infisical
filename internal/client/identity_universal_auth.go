package infisicalclient

import (
	"fmt"
	"net/http"
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
		return IdentityUniversalAuth{}, fmt.Errorf("GetIdentityUniversalAuth: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return IdentityUniversalAuth{}, fmt.Errorf("GetIdentityUniversalAuth: Unsuccessful response. [response=%v]", string(response.Body()))
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
		return IdentityUniversalAuth{}, fmt.Errorf("CreateIdentityUniversalAuth: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return IdentityUniversalAuth{}, fmt.Errorf("CreateIdentityUniversalAuth: Unsuccessful response. [response=%s]", string(response.Body()))
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
		return IdentityUniversalAuth{}, fmt.Errorf("UpdateIdentityUniversalAuth: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return IdentityUniversalAuth{}, fmt.Errorf("UpdateIdentityUniversalAuth: Unsuccessful response. [response=%s]", string(response.Body()))
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
		return IdentityUniversalAuth{}, fmt.Errorf("RevokeIdentityUniversalAuth: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return IdentityUniversalAuth{}, fmt.Errorf("RevokeIdentityUniversalAuth: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body.UniversalAuth, nil
}
