package infisicalclient

import (
	"fmt"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationGetCertManagerIdentity    = "CallGetCertManagerIdentity"
	operationAddCertManagerIdentity    = "CallAddCertManagerIdentity"
	operationUpdateCertManagerIdentity = "CallUpdateCertManagerIdentity"
	operationRemoveCertManagerIdentity = "CallRemoveCertManagerIdentity"
)

func (client Client) GetCertManagerIdentity(request GetCertManagerIdentityRequest) (GetCertManagerIdentityResponse, error) {
	var identityResponse GetCertManagerIdentityResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&identityResponse).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/cert-manager/access/identities/%s", request.IdentityId))

	if err != nil {
		return GetCertManagerIdentityResponse{}, errors.NewGenericRequestError(operationGetCertManagerIdentity, err)
	}

	if response.IsError() {
		if response.StatusCode() == 404 || response.StatusCode() == 422 {
			return GetCertManagerIdentityResponse{}, ErrNotFound
		}
		return GetCertManagerIdentityResponse{}, errors.NewAPIErrorWithResponse(operationGetCertManagerIdentity, response, nil)
	}

	return identityResponse, nil
}

func (client Client) AddCertManagerIdentity(request AddCertManagerIdentityRequest) (AddCertManagerIdentityResponse, error) {
	var identityResponse AddCertManagerIdentityResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&identityResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(fmt.Sprintf("api/v1/cert-manager/access/identities/%s", request.IdentityId))

	if err != nil {
		return AddCertManagerIdentityResponse{}, errors.NewGenericRequestError(operationAddCertManagerIdentity, err)
	}

	if response.IsError() {
		return AddCertManagerIdentityResponse{}, errors.NewAPIErrorWithResponse(operationAddCertManagerIdentity, response, nil)
	}

	return identityResponse, nil
}

func (client Client) UpdateCertManagerIdentity(request UpdateCertManagerIdentityRequest) (UpdateCertManagerIdentityResponse, error) {
	var identityResponse UpdateCertManagerIdentityResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&identityResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/cert-manager/access/identities/%s", request.IdentityId))

	if err != nil {
		return UpdateCertManagerIdentityResponse{}, errors.NewGenericRequestError(operationUpdateCertManagerIdentity, err)
	}

	if response.IsError() {
		return UpdateCertManagerIdentityResponse{}, errors.NewAPIErrorWithResponse(operationUpdateCertManagerIdentity, response, nil)
	}

	return identityResponse, nil
}

func (client Client) RemoveCertManagerIdentity(request RemoveCertManagerIdentityRequest) (RemoveCertManagerIdentityResponse, error) {
	var identityResponse RemoveCertManagerIdentityResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&identityResponse).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/cert-manager/access/identities/%s", request.IdentityId))

	if err != nil {
		return RemoveCertManagerIdentityResponse{}, errors.NewGenericRequestError(operationRemoveCertManagerIdentity, err)
	}

	if response.IsError() {
		return RemoveCertManagerIdentityResponse{}, errors.NewAPIErrorWithResponse(operationRemoveCertManagerIdentity, response, nil)
	}

	return identityResponse, nil
}
