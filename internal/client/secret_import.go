package infisicalclient

import (
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationGetSecretImportByID = "CallGetSecretImportByID"
	operationCreateSecretImport  = "CallCreateSecretImport"
	operationUpdateSecretImport  = "CallUpdateSecretImport"
	operationDeleteSecretImport  = "CallDeleteSecretImport"
)

func (client Client) GetSecretImportByID(request GetSecretImportByIDRequest) (GetSecretImportByIDResponse, error) {
	var body GetSecretImportByIDResponse

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT)

	response, err := httpRequest.Get("api/v2/secret-imports/" + request.ID)

	if err != nil {
		return GetSecretImportByIDResponse{}, errors.NewGenericRequestError(operationGetSecretImportByID, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return GetSecretImportByIDResponse{}, ErrNotFound
		}
		return GetSecretImportByIDResponse{}, errors.NewAPIErrorWithResponse(operationGetSecretImportByID, response, nil)
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
		Post("api/v2/secret-imports")

	if err != nil {
		return CreateSecretImportResponse{}, errors.NewGenericRequestError(operationCreateSecretImport, err)
	}

	if response.IsError() {
		return CreateSecretImportResponse{}, errors.NewAPIErrorWithResponse(operationCreateSecretImport, response, nil)
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
		Patch("api/v2/secret-imports/" + request.ID)

	if err != nil {
		return UpdateSecretImportResponse{}, errors.NewGenericRequestError(operationUpdateSecretImport, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return UpdateSecretImportResponse{}, NewNotFoundError("SecretImport", request.SecretPath)
		}

		return UpdateSecretImportResponse{}, errors.NewAPIErrorWithResponse(operationUpdateSecretImport, response, nil)
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
		Delete("api/v2/secret-imports/" + request.ID)

	if err != nil {
		return DeleteSecretImportResponse{}, errors.NewGenericRequestError(operationDeleteSecretImport, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return DeleteSecretImportResponse{}, NewNotFoundError("SecretImport", request.SecretPath)
		}

		return DeleteSecretImportResponse{}, errors.NewAPIErrorWithResponse(operationDeleteSecretImport, response, nil)
	}

	return body, nil
}
