package infisicalclient

import (
	"fmt"
	"net/http"
)

func (client Client) GetIdentityKubernetesAuth(request GetIdentityKubernetesAuthRequest) (IdentityKubernetesAuth, error) {
	var body GetIdentityKubernetesAuthResponse

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT)

	response, err := httpRequest.Get("api/v1/auth/kubernetes-auth/identities/" + request.IdentityID)

	if response.StatusCode() == http.StatusNotFound {
		return IdentityKubernetesAuth{}, ErrNotFound
	}

	if err != nil {
		return IdentityKubernetesAuth{}, fmt.Errorf("GetIdentityKubernetesAuth: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return IdentityKubernetesAuth{}, fmt.Errorf("GetIdentityKubernetesAuth: Unsuccessful response. [response=%v]", string(response.Body()))
	}

	return body.IdentityKubernetesAuth, nil
}

func (client Client) CreateIdentityKubernetesAuth(request CreateIdentityKubernetesAuthRequest) (IdentityKubernetesAuth, error) {
	var body CreateIdentityKubernetesAuthResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/auth/kubernetes-auth/identities/" + request.IdentityID)

	if err != nil {
		return IdentityKubernetesAuth{}, fmt.Errorf("CreateIdentityKubernetesAuth: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return IdentityKubernetesAuth{}, fmt.Errorf("CreateIdentityKubernetesAuth: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body.IdentityKubernetesAuth, nil
}

func (client Client) UpdateIdentityKubernetesAuth(request UpdateIdentityKubernetesAuthRequest) (IdentityKubernetesAuth, error) {
	var body UpdateIdentityKubernetesAuthResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch("api/v1/auth/kubernetes-auth/identities/" + request.IdentityID)

	if err != nil {
		return IdentityKubernetesAuth{}, fmt.Errorf("UpdateIdentityKubernetesAuth: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return IdentityKubernetesAuth{}, fmt.Errorf("UpdateIdentityKubernetesAuth: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body.IdentityKubernetesAuth, nil
}

func (client Client) RevokeIdentityKubernetesAuth(request RevokeIdentityKubernetesAuthRequest) (IdentityKubernetesAuth, error) {
	var body RevokeIdentityKubernetesAuthResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Delete("api/v1/auth/kubernetes-auth/identities/" + request.IdentityID)

	if err != nil {
		return IdentityKubernetesAuth{}, fmt.Errorf("RevokeIdentityKubernetesAuth: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return IdentityKubernetesAuth{}, fmt.Errorf("RevokeIdentityKubernetesAuth: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body.IdentityKubernetesAuth, nil
}
