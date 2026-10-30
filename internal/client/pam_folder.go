package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationCreatePamFolder = "CallCreatePamFolder"
	operationGetPamFolder    = "CallGetPamFolder"
	operationUpdatePamFolder = "CallUpdatePamFolder"
	operationDeletePamFolder = "CallDeletePamFolder"
)

func (client Client) CreatePamFolder(request CreatePamFolderRequest) (PamFolder, error) {
	var body PamFolderResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/pam/folders")

	if err != nil {
		return PamFolder{}, errors.NewGenericRequestError(operationCreatePamFolder, err)
	}

	if response.IsError() {
		return PamFolder{}, errors.NewAPIErrorWithResponse(operationCreatePamFolder, response, nil)
	}

	return body.Folder, nil
}

func (client Client) GetPamFolder(request GetPamFolderRequest) (PamFolder, error) {
	var body PamFolderResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/pam/folders/%s", request.ID))

	if err != nil {
		return PamFolder{}, errors.NewGenericRequestError(operationGetPamFolder, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return PamFolder{}, ErrNotFound
		}
		return PamFolder{}, errors.NewAPIErrorWithResponse(operationGetPamFolder, response, nil)
	}

	return body.Folder, nil
}

func (client Client) UpdatePamFolder(request UpdatePamFolderRequest) (PamFolder, error) {
	var body PamFolderResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/pam/folders/%s", request.ID))

	if err != nil {
		return PamFolder{}, errors.NewGenericRequestError(operationUpdatePamFolder, err)
	}

	if response.IsError() {
		return PamFolder{}, errors.NewAPIErrorWithResponse(operationUpdatePamFolder, response, nil)
	}

	return body.Folder, nil
}

func (client Client) DeletePamFolder(request DeletePamFolderRequest) error {
	response, err := client.Config.HttpClient.
		R().
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/pam/folders/%s", request.ID))

	if err != nil {
		return errors.NewGenericRequestError(operationDeletePamFolder, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return ErrNotFound
		}
		return errors.NewAPIErrorWithResponse(operationDeletePamFolder, response, nil)
	}

	return nil
}
