package infisicalclient

import (
	"fmt"
	"net/http"
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
		return IdentityGcpAuth{}, fmt.Errorf("GetIdentityGcpAuth: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return IdentityGcpAuth{}, fmt.Errorf("GetIdentityGcpAuth: Unsuccessful response. [response=%v]", string(response.Body()))
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
		return IdentityGcpAuth{}, fmt.Errorf("CreateIdentityGcpAuth: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return IdentityGcpAuth{}, fmt.Errorf("CreateIdentityGcpAuth: Unsuccessful response. [response=%s]", string(response.Body()))
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
		return IdentityGcpAuth{}, fmt.Errorf("UpdateIdentityGcpAuth: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return IdentityGcpAuth{}, fmt.Errorf("UpdateIdentityGcpAuth: Unsuccessful response. [response=%s]", string(response.Body()))
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
		return IdentityGcpAuth{}, fmt.Errorf("RevokeIdentityGcpAuth: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return IdentityGcpAuth{}, fmt.Errorf("RevokeIdentityGcpAuth: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body.IdentityGcpAuth, nil
}
