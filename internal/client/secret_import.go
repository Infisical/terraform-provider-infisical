package infisicalclient

import (
	"fmt"
	"net/http"
)

// Workaround to getSecretImportById API call.
func findSecretImportByID(secretImports []SecretImport, id string) (GetSecretImportByIDResponse, error) {
	for _, secretImport := range secretImports {
		if secretImport.ID == id {
			return GetSecretImportByIDResponse{SecretImport: secretImport}, nil
		}
	}

	return GetSecretImportByIDResponse{}, NewNotFoundError("SecretImport", id)
}

func (client Client) GetSecretImportByID(request GetSecretImportByIDRequest) (GetSecretImportByIDResponse, error) {
	var body ListSecretImportResponse

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetQueryParam("workspaceId", request.ProjectID).
		SetQueryParam("environment", request.Environment).
		SetQueryParam("path", request.SecretPath)

	response, err := httpRequest.Get("api/v1/secret-imports")

	if err != nil {
		return GetSecretImportByIDResponse{}, fmt.Errorf("GetSecretImportByID: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return GetSecretImportByIDResponse{}, fmt.Errorf("GetSecretImportByID: Unsuccessful response. [response=%v]", string(response.Body()))
	}

	return findSecretImportByID(body.SecretImports, request.ID)

}

func (client Client) GetSecretImportList(request ListSecretImportRequest) (ListSecretImportResponse, error) {
	var body ListSecretImportResponse

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetQueryParam("workspaceId", request.ProjectID).
		SetQueryParam("environment", request.Environment).
		SetQueryParam("path", request.SecretPath)

	response, err := httpRequest.Get("api/v1/secret-imports")

	if err != nil {
		return ListSecretImportResponse{}, fmt.Errorf("ListSecretImport: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return ListSecretImportResponse{}, fmt.Errorf("ListSecretImport: Unsuccessful response. [response=%v]", string(response.Body()))
	}

	return body, nil
}

func (client Client) CreateSecretImport(request CreateSecretImportRequest) (CreateSecretImportResponse, error) {
	var body CreateSecretImportResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/secret-imports")

	if err != nil {
		return CreateSecretImportResponse{}, fmt.Errorf("CallCreateSecretImport: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return CreateSecretImportResponse{}, fmt.Errorf("CallCreateSecretImport: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body, nil
}

func (client Client) UpdateSecretImport(request UpdateSecretImportRequest) (UpdateSecretImportResponse, error) {
	var body UpdateSecretImportResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch("api/v1/secret-imports/" + request.ID)

	if err != nil {
		return UpdateSecretImportResponse{}, fmt.Errorf("CallUpdateSecretImport: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return UpdateSecretImportResponse{}, NewNotFoundError("SecretImport", request.SecretPath)
		}

		return UpdateSecretImportResponse{}, fmt.Errorf("CallUpdateSecretImport: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body, nil
}

func (client Client) DeleteSecretImport(request DeleteSecretImportRequest) (DeleteSecretImportResponse, error) {
	var body DeleteSecretImportResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Delete("api/v1/secret-imports/" + request.ID)

	if err != nil {
		return DeleteSecretImportResponse{}, fmt.Errorf("CallDeleteSecretImport: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return DeleteSecretImportResponse{}, NewNotFoundError("SecretImport", request.SecretPath)
		}

		return DeleteSecretImportResponse{}, fmt.Errorf("CallDeleteSecretImport: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body, nil
}
