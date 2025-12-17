package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

type ExternalKmsProvider string

const (
	ExternalKmsProviderAWS ExternalKmsProvider = "aws"
)

const (
	operationCreateExternalKms  = "CallCreateExternalKms"
	operationGetExternalKmsById = "CallGetExternalKmsById"
	operationUpdateExternalKms  = "CallUpdateExternalKms"
	operationDeleteExternalKms  = "CallDeleteExternalKms"
)

func (client Client) CreateExternalKms(request CreateExternalKmsRequest) (ExternalKmsWithKey, error) {
	var body ExternalKmsWithKey
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/external-kms/" + string(request.Provider))

	if err != nil {
		return ExternalKmsWithKey{}, errors.NewGenericRequestError(operationCreateExternalKms, err)
	}

	if response.IsError() {
		return ExternalKmsWithKey{}, errors.NewAPIErrorWithResponse(operationCreateExternalKms, response, nil)
	}

	return body, nil
}

func (client Client) GetExternalKmsById(request GetExternalKmsByIdRequest) (ExternalKmsWithKey, error) {
	var body ExternalKmsWithKey
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/external-kms/%s/%s", request.Provider, request.ID))

	if err != nil {
		return ExternalKmsWithKey{}, errors.NewGenericRequestError(operationGetExternalKmsById, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return ExternalKmsWithKey{}, ErrNotFound
		}
		return ExternalKmsWithKey{}, errors.NewAPIErrorWithResponse(operationGetExternalKmsById, response, nil)
	}

	return body, nil
}

func (client Client) UpdateExternalKms(request UpdateExternalKmsRequest) (ExternalKmsWithKey, error) {
	var body ExternalKmsWithKey
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/external-kms/%s/%s", request.Provider, request.ID))

	if err != nil {
		return ExternalKmsWithKey{}, errors.NewGenericRequestError(operationUpdateExternalKms, err)
	}

	if response.IsError() {
		return ExternalKmsWithKey{}, errors.NewAPIErrorWithResponse(operationUpdateExternalKms, response, nil)
	}

	return body, nil
}

func (client Client) DeleteExternalKms(request DeleteExternalKmsRequest) (ExternalKmsWithKey, error) {
	var body ExternalKmsWithKey
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/external-kms/%s/%s", request.Provider, request.ID))

	if err != nil {
		return ExternalKmsWithKey{}, errors.NewGenericRequestError(operationDeleteExternalKms, err)
	}

	if response.IsError() {
		return ExternalKmsWithKey{}, errors.NewAPIErrorWithResponse(operationDeleteExternalKms, response, nil)
	}

	return body, nil
}
