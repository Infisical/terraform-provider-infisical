package infisicalclient

import (
	"fmt"
)

func (client Client) GetSecretFolderByID(request GetSecretFolderByIDRequest) (GetSecretFolderByIDResponse, error) {
	var body GetSecretFolderByIDResponse

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT)

	response, err := httpRequest.Get("api/v1/folders/" + request.ID)

	if err != nil {
		return GetSecretFolderByIDResponse{}, fmt.Errorf("GetSecretFolderByID: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return GetSecretFolderByIDResponse{}, fmt.Errorf("GetSecretFolderByID: Unsuccessful response. [response=%v]", string(response.Body()))
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
		return ListSecretFolderResponse{}, fmt.Errorf("ListSecretFolder: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return ListSecretFolderResponse{}, fmt.Errorf("ListSecretFolder: Unsuccessful response. [response=%v]", string(response.Body()))
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
		return CreateSecretFolderResponse{}, fmt.Errorf("CallCreateSecretFolder: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return CreateSecretFolderResponse{}, fmt.Errorf("CallCreateSecretFolder: Unsuccessful response. [response=%s]", string(response.Body()))
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
		return UpdateSecretFolderResponse{}, fmt.Errorf("CallUpdateSecretFolder: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return UpdateSecretFolderResponse{}, fmt.Errorf("CallUpdateSecretFolder: Unsuccessful response. [response=%s]", string(response.Body()))
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
		return DeleteSecretFolderResponse{}, fmt.Errorf("CallDeleteSecretFolder: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return DeleteSecretFolderResponse{}, fmt.Errorf("CallDeleteSecretFolder: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body, nil
}
