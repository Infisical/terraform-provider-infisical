package infisicalclient

import (
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationGetIdentityAwsAuth    = "CallGetIdentityAwsAuth"
	operationCreateIdentityAwsAuth = "CallCreateIdentityAwsAuth"
	operationUpdateIdentityAwsAuth = "CallUpdateIdentityAwsAuth"
	operationRevokeIdentityAwsAuth = "CallRevokeIdentityAwsAuth"
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
		return IdentityAwsAuth{}, errors.NewGenericRequestError(operationGetIdentityAwsAuth, err)
	}

	if response.IsError() {
		return IdentityAwsAuth{}, errors.NewAPIErrorWithResponse(operationGetIdentityAwsAuth, response, nil)
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
		return IdentityAwsAuth{}, errors.NewGenericRequestError(operationCreateIdentityAwsAuth, err)
	}

	if response.IsError() {
		return IdentityAwsAuth{}, errors.NewAPIErrorWithResponse(operationCreateIdentityAwsAuth, response, nil)
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
		return IdentityAwsAuth{}, errors.NewGenericRequestError(operationUpdateIdentityAwsAuth, err)
	}

	if response.IsError() {
		return IdentityAwsAuth{}, errors.NewAPIErrorWithResponse(operationUpdateIdentityAwsAuth, response, nil)
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
		return IdentityAwsAuth{}, errors.NewGenericRequestError(operationRevokeIdentityAwsAuth, err)
	}

	if response.IsError() {
		return IdentityAwsAuth{}, errors.NewAPIErrorWithResponse(operationRevokeIdentityAwsAuth, response, nil)
	}

	return body.IdentityAwsAuth, nil
}
