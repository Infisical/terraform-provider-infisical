package infisicalclient

import (
	"fmt"
	"net/http"
	"net/url"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationGetSecretFolderByID   = "CallGetSecretFolderByID"
	operationGetSecretFolderList   = "CallGetSecretFolderList"
	operationCreateSecretFolder    = "CallCreateSecretFolder"
	operationUpdateSecretFolder    = "CallUpdateSecretFolder"
	operationDeleteSecretFolder    = "CallDeleteSecretFolder"
	operationGetSecretFolderByPath = "CallGetSecretFolderByPath"
)

func (client Client) GetSecretFolderByID(request GetSecretFolderByIDRequest) (GetSecretFolderByIDResponse, error) {
	var body GetSecretFolderByIDResponse

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT)

	response, err := httpRequest.Get("api/v1/folders/" + request.ID)

	if err != nil {
		return GetSecretFolderByIDResponse{}, errors.NewGenericRequestError(operationGetSecretFolderByID, err)
	}

	if response.IsError() {
		return GetSecretFolderByIDResponse{}, errors.NewAPIErrorWithResponse(operationGetSecretFolderByID, response, nil)
	}

	return body, nil
}

func (client Client) GetSecretFolderList(request ListSecretFolderRequest) (ListSecretFolderResponse, error) {
	var body ListSecretFolderResponse

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetQueryParam("workspaceId", request.ProjectID).
		SetQueryParam("environment", request.Environment).
		SetQueryParam("path", request.SecretPath)

	response, err := httpRequest.Get("api/v1/folders")

	if err != nil {
		return ListSecretFolderResponse{}, errors.NewGenericRequestError(operationGetSecretFolderList, err)
	}

	if response.IsError() {
		return ListSecretFolderResponse{}, errors.NewAPIErrorWithResponse(operationGetSecretFolderList, response, nil)
	}

	return body, nil
}

func (client Client) CreateSecretFolder(request CreateSecretFolderRequest) (CreateSecretFolderResponse, error) {
	var body CreateSecretFolderResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/folders")

	if err != nil {
		return CreateSecretFolderResponse{}, errors.NewGenericRequestError(operationCreateSecretFolder, err)
	}

	if response.IsError() {
		return CreateSecretFolderResponse{}, errors.NewAPIErrorWithResponse(operationCreateSecretFolder, response, nil)
	}

	return body, nil
}

func (client Client) UpdateSecretFolder(request UpdateSecretFolderRequest) (UpdateSecretFolderResponse, error) {
	var body UpdateSecretFolderResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch("api/v1/folders/" + request.ID)

	if err != nil {
		return UpdateSecretFolderResponse{}, errors.NewGenericRequestError(operationUpdateSecretFolder, err)
	}

	if response.IsError() {
		return UpdateSecretFolderResponse{}, errors.NewAPIErrorWithResponse(operationUpdateSecretFolder, response, nil)
	}

	return body, nil
}

func (client Client) DeleteSecretFolder(request DeleteSecretFolderRequest) (DeleteSecretFolderResponse, error) {
	var body DeleteSecretFolderResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Delete("api/v1/folders/" + request.ID)

	if err != nil {
		return DeleteSecretFolderResponse{}, errors.NewGenericRequestError(operationDeleteSecretFolder, err)
	}

	if response.IsError() {
		return DeleteSecretFolderResponse{}, errors.NewAPIErrorWithResponse(operationDeleteSecretFolder, response, nil)
	}

	return body, nil
}

func (client Client) GetFolderByPath(request GetSecretFolderByPathRequest) (GetSecretFolderByPathResponse, error) {
	var body GetSecretFolderByPathResponse

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT)
	response, err := httpRequest.Get(fmt.Sprintf("api/v1/folders/%s/%s/%s", request.ProjectID, request.Environment, url.PathEscape(request.SecretPath)))

	if err != nil {
		return GetSecretFolderByPathResponse{}, errors.NewGenericRequestError(operationGetSecretFolderByPath, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return GetSecretFolderByPathResponse{}, ErrNotFound
		}
		return GetSecretFolderByPathResponse{}, errors.NewAPIErrorWithResponse(operationGetSecretFolderByPath, response, nil)
	}

	return body, nil
}
