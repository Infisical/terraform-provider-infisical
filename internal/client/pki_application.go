package infisicalclient

import (
	"fmt"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationCreatePkiApplication = "CallCreatePkiApplication"
	operationGetPkiApplication    = "CallGetPkiApplication"
	operationUpdatePkiApplication = "CallUpdatePkiApplication"
	operationDeletePkiApplication = "CallDeletePkiApplication"
)

func (client Client) CreatePkiApplication(request CreatePkiApplicationRequest) (CreatePkiApplicationResponse, error) {
	var applicationResponse CreatePkiApplicationResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&applicationResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/cert-manager/applications")

	if err != nil {
		return CreatePkiApplicationResponse{}, errors.NewGenericRequestError(operationCreatePkiApplication, err)
	}

	if response.IsError() {
		return CreatePkiApplicationResponse{}, errors.NewAPIErrorWithResponse(operationCreatePkiApplication, response, nil)
	}

	return applicationResponse, nil
}

func (client Client) GetPkiApplication(request GetPkiApplicationRequest) (GetPkiApplicationResponse, error) {
	var applicationResponse GetPkiApplicationResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&applicationResponse).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/cert-manager/applications/%s", request.ApplicationId))

	if err != nil {
		return GetPkiApplicationResponse{}, errors.NewGenericRequestError(operationGetPkiApplication, err)
	}

	if response.IsError() {
		if response.StatusCode() == 404 || response.StatusCode() == 422 {
			return GetPkiApplicationResponse{}, ErrNotFound
		}
		return GetPkiApplicationResponse{}, errors.NewAPIErrorWithResponse(operationGetPkiApplication, response, nil)
	}

	return applicationResponse, nil
}

func (client Client) UpdatePkiApplication(request UpdatePkiApplicationRequest) (UpdatePkiApplicationResponse, error) {
	var applicationResponse UpdatePkiApplicationResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&applicationResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/cert-manager/applications/%s", request.ApplicationId))

	if err != nil {
		return UpdatePkiApplicationResponse{}, errors.NewGenericRequestError(operationUpdatePkiApplication, err)
	}

	if response.IsError() {
		return UpdatePkiApplicationResponse{}, errors.NewAPIErrorWithResponse(operationUpdatePkiApplication, response, nil)
	}

	return applicationResponse, nil
}

func (client Client) DeletePkiApplication(request DeletePkiApplicationRequest) (DeletePkiApplicationResponse, error) {
	var applicationResponse DeletePkiApplicationResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&applicationResponse).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/cert-manager/applications/%s", request.ApplicationId))

	if err != nil {
		return DeletePkiApplicationResponse{}, errors.NewGenericRequestError(operationDeletePkiApplication, err)
	}

	if response.IsError() {
		return DeletePkiApplicationResponse{}, errors.NewAPIErrorWithResponse(operationDeletePkiApplication, response, nil)
	}

	return applicationResponse, nil
}
