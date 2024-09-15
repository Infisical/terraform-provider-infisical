package infisicalclient

import (
	"fmt"
	"net/http"
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
		return IdentityOidcAuth{}, fmt.Errorf("CreateIdentityOidcAuth: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return IdentityOidcAuth{}, fmt.Errorf("CreateIdentityOidcAuth: Unsuccessful response. [response=%s]", string(response.Body()))
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
		return IdentityOidcAuth{}, fmt.Errorf("GetIdentityOidcAuth: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return IdentityOidcAuth{}, fmt.Errorf("GetIdentityOidcAuth: Unsuccessful response. [response=%v]", string(response.Body()))
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
		return IdentityOidcAuth{}, fmt.Errorf("UpdateIdentityOidcAuth: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return IdentityOidcAuth{}, fmt.Errorf("UpdateIdentityOidcAuth: Unsuccessful response. [response=%s]", string(response.Body()))
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
		return IdentityOidcAuth{}, fmt.Errorf("RevokeIdentityOidcAuth: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return IdentityOidcAuth{}, fmt.Errorf("RevokeIdentityOidcAuth: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body.IdentityOidcAuth, nil
}
