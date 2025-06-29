package infisicalclient

import (
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationGetIdentityKubernetesAuth    = "CallGetIdentityKubernetesAuth"
	operationCreateIdentityKubernetesAuth = "CallCreateIdentityKubernetesAuth"
	operationUpdateIdentityKubernetesAuth = "CallUpdateIdentityKubernetesAuth"
	operationRevokeIdentityKubernetesAuth = "CallRevokeIdentityKubernetesAuth"
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
		return IdentityKubernetesAuth{}, errors.NewGenericRequestError(operationGetIdentityKubernetesAuth, err)
	}

	if response.IsError() {
		return IdentityKubernetesAuth{}, errors.NewAPIErrorWithResponse(operationGetIdentityKubernetesAuth, response, nil)
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
		return IdentityKubernetesAuth{}, errors.NewGenericRequestError(operationCreateIdentityKubernetesAuth, err)
	}

	if response.IsError() {
		return IdentityKubernetesAuth{}, errors.NewAPIErrorWithResponse(operationCreateIdentityKubernetesAuth, response, nil)
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
		return IdentityKubernetesAuth{}, errors.NewGenericRequestError(operationUpdateIdentityKubernetesAuth, err)
	}

	if response.IsError() {
		return IdentityKubernetesAuth{}, errors.NewAPIErrorWithResponse(operationUpdateIdentityKubernetesAuth, response, nil)
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
		return IdentityKubernetesAuth{}, errors.NewGenericRequestError(operationRevokeIdentityKubernetesAuth, err)
	}

	if response.IsError() {
		return IdentityKubernetesAuth{}, errors.NewAPIErrorWithResponse(operationRevokeIdentityKubernetesAuth, response, nil)
	}

	return body.IdentityKubernetesAuth, nil
}
