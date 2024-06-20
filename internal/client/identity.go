package infisicalclient

import (
	"fmt"
	"net/http"
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
		return OrgIdentity{}, fmt.Errorf("CallGetIdentity: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return OrgIdentity{}, fmt.Errorf("CallGetIdentity: Unsuccessful response. [response=%v]", string(response.Body()))
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
		return CreateIdentityResponse{}, fmt.Errorf("CallCreateIdentity: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return CreateIdentityResponse{}, fmt.Errorf("CallCreateIdentity: Unsuccessful response. [response=%s]", string(response.Body()))
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
		return UpdateIdentityResponse{}, fmt.Errorf("CallUpdateIdentity: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return UpdateIdentityResponse{}, fmt.Errorf("CallUpdateIdentity: Unsuccessful response. [response=%s]", string(response.Body()))
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
		return DeleteIdentityResponse{}, fmt.Errorf("CallDeleteIdentity: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return DeleteIdentityResponse{}, fmt.Errorf("CallDeleteIdentity: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body, nil
}
