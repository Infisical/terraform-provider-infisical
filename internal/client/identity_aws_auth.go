package infisicalclient

import (
	"fmt"
	"net/http"
)

func (client Client) GetIdentityAwsAuth(request GetIdentityAwsAuthRequest) (IdentityAwsAuth, error) {
	var body GetIdentityAwsAuthResponse

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT)

	response, err := httpRequest.Get("api/v1/auth/aws-auth/identities/" + request.IdentityID)

	if response.StatusCode() == http.StatusNotFound {
		return IdentityAwsAuth{}, ErrNotFound
	}

	if err != nil {
		return IdentityAwsAuth{}, fmt.Errorf("GetIdentityAwsAuth: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return IdentityAwsAuth{}, fmt.Errorf("GetIdentityAwsAuth: Unsuccessful response. [response=%v]", string(response.Body()))
	}

	return body.IdentityAwsAuth, nil
}

func (client Client) CreateIdentityAwsAuth(request CreateIdentityAwsAuthRequest) (IdentityAwsAuth, error) {
	var body CreateIdentityAwsAuthResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/auth/aws-auth/identities/" + request.IdentityID)

	if err != nil {
		return IdentityAwsAuth{}, fmt.Errorf("CreateIdentityAwsAuth: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return IdentityAwsAuth{}, fmt.Errorf("CreateIdentityAwsAuth: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body.IdentityAwsAuth, nil
}

func (client Client) UpdateIdentityAwsAuth(request UpdateIdentityAwsAuthRequest) (IdentityAwsAuth, error) {
	var body UpdateIdentityAwsAuthResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch("api/v1/auth/aws-auth/identities/" + request.IdentityID)

	if err != nil {
		return IdentityAwsAuth{}, fmt.Errorf("UpdateIdentityAwsAuth: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return IdentityAwsAuth{}, fmt.Errorf("UpdateIdentityAwsAuth: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body.IdentityAwsAuth, nil
}

func (client Client) RevokeIdentityAwsAuth(request RevokeIdentityAwsAuthRequest) (IdentityAwsAuth, error) {
	var body RevokeIdentityAwsAuthResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Delete("api/v1/auth/aws-auth/identities/" + request.IdentityID)

	if err != nil {
		return IdentityAwsAuth{}, fmt.Errorf("RevokeIdentityAwsAuth: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return IdentityAwsAuth{}, fmt.Errorf("RevokeIdentityAwsAuth: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body.IdentityAwsAuth, nil
}
