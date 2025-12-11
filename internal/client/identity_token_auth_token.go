package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationGetIdentityTokenAuthToken    = "CallGetIdentityTokenAuthToken"
	operationCreateIdentityTokenAuthToken = "CallCreateIdentityTokenAuthToken"
	operationUpdateIdentityTokenAuthToken = "CallUpdateIdentityTokenAuthToken"
	operationRevokeIdentityTokenAuthToken = "CallRevokeIdentityTokenAuthToken"
)

func (client Client) GetIdentityTokenAuthToken(request GetIdentityTokenAuthTokenRequest) (IdentityTokenAuthToken, error) {
	var body GetIdentityTokenAuthTokenResponse

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT)

	response, err := httpRequest.Get(fmt.Sprintf("api/v1/auth/token-auth/tokens/%s", request.TokenID))

	if response.StatusCode() == http.StatusNotFound {
		return IdentityTokenAuthToken{}, ErrNotFound
	}

	if err != nil {
		return IdentityTokenAuthToken{}, errors.NewGenericRequestError(operationGetIdentityTokenAuthToken, err)
	}

	if response.IsError() {
		return IdentityTokenAuthToken{}, errors.NewAPIErrorWithResponse(operationGetIdentityTokenAuthToken, response, nil)
	}

	return body.Token, nil
}

func (client Client) CreateIdentityTokenAuthToken(request CreateIdentityTokenAuthTokenRequest) (CreateIdentityTokenAuthTokenResponse, error) {
	var body CreateIdentityTokenAuthTokenResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/auth/token-auth/identities/" + request.IdentityID + "/tokens")

	if err != nil {
		return CreateIdentityTokenAuthTokenResponse{}, errors.NewGenericRequestError(operationCreateIdentityTokenAuthToken, err)
	}

	if response.IsError() {
		return CreateIdentityTokenAuthTokenResponse{}, errors.NewAPIErrorWithResponse(operationCreateIdentityTokenAuthToken, response, nil)
	}

	return body, nil
}

func (client Client) UpdateIdentityTokenAuthToken(request UpdateIdentityTokenAuthTokenRequest) (IdentityTokenAuthToken, error) {
	var body UpdateIdentityTokenAuthTokenResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch("api/v1/auth/token-auth/tokens/" + request.TokenID)

	if err != nil {
		return IdentityTokenAuthToken{}, errors.NewGenericRequestError(operationUpdateIdentityTokenAuthToken, err)
	}

	if response.IsError() {
		return IdentityTokenAuthToken{}, errors.NewAPIErrorWithResponse(operationUpdateIdentityTokenAuthToken, response, nil)
	}

	return body.Token, nil
}

func (client Client) RevokeIdentityTokenAuthToken(request RevokeIdentityTokenAuthTokenRequest) (IdentityTokenAuthToken, error) {
	var body RevokeIdentityTokenAuthTokenResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/auth/token-auth/tokens/" + request.TokenID + "/revoke")

	if err != nil {
		return IdentityTokenAuthToken{}, errors.NewGenericRequestError(operationRevokeIdentityTokenAuthToken, err)
	}

	if response.IsError() {
		return IdentityTokenAuthToken{}, errors.NewAPIErrorWithResponse(operationRevokeIdentityTokenAuthToken, response, nil)
	}

	return body.Token, nil
}
