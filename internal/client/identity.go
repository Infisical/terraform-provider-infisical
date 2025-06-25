package infisicalclient

import (
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationGetIdentity    = "CallGetIdentity"
	operationCreateIdentity = "CallCreateIdentity"
	operationUpdateIdentity = "CallUpdateIdentity"
	operationDeleteIdentity = "CallDeleteIdentity"
)

func (client Client) GetIdentity(request GetIdentityRequest) (OrgIdentity, error) {
	var body GetIdentityResponse

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT)

	response, err := httpRequest.Get("api/v1/identities/" + request.IdentityID)

	if response.StatusCode() == http.StatusNotFound {
		return OrgIdentity{}, ErrNotFound
	}

	if err != nil {
		return OrgIdentity{}, errors.NewGenericRequestError(operationGetIdentity, err)
	}

	if response.IsError() {
		return OrgIdentity{}, errors.NewAPIErrorWithResponse(operationGetIdentity, response, nil)
	}

	return body.Identity, nil
}

func (client Client) CreateIdentity(request CreateIdentityRequest) (CreateIdentityResponse, error) {
	var body CreateIdentityResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/identities")

	if err != nil {
		return CreateIdentityResponse{}, errors.NewGenericRequestError(operationCreateIdentity, err)
	}

	if response.IsError() {
		return CreateIdentityResponse{}, errors.NewAPIErrorWithResponse(operationCreateIdentity, response, nil)
	}

	return body, nil
}

func (client Client) UpdateIdentity(request UpdateIdentityRequest) (UpdateIdentityResponse, error) {
	var body UpdateIdentityResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch("api/v1/identities/" + request.IdentityID)

	if err != nil {
		return UpdateIdentityResponse{}, errors.NewGenericRequestError(operationUpdateIdentity, err)
	}

	if response.IsError() {
		return UpdateIdentityResponse{}, errors.NewAPIErrorWithResponse(operationUpdateIdentity, response, nil)
	}

	return body, nil
}

func (client Client) DeleteIdentity(request DeleteIdentityRequest) (DeleteIdentityResponse, error) {
	var body DeleteIdentityResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Delete("api/v1/identities/" + request.IdentityID)

	if err != nil {
		return DeleteIdentityResponse{}, errors.NewGenericRequestError(operationDeleteIdentity, err)
	}

	if response.IsError() {
		return DeleteIdentityResponse{}, errors.NewAPIErrorWithResponse(operationDeleteIdentity, response, nil)
	}

	return body, nil
}
