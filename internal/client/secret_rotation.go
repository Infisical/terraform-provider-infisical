package infisicalclient

import (
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

type SecretRotationProvider string

const (
	SecretRotationProviderMySqlCredentials  SecretRotationProvider = "mysql-credentials"
	SecretRotationProviderAzureClientSecret SecretRotationProvider = "azure-client-secret"
)

const (
	operationCreateSecretRotation  = "CallCreateSecretRotation"
	operationGetSecretRotationById = "CallGetSecretRotationById"
	operationUpdateSecretRotation  = "CallUpdateSecretRotation"
	operationDeleteSecretRotation  = "CallDeleteSecretRotation"
)

func (client Client) CreateSecretRotation(request CreateSecretRotationRequest) (SecretRotation, error) {
	var body CreateSecretRotationResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v2/secret-rotations/" + string(request.Provider))

	if err != nil {
		return SecretRotation{}, errors.NewGenericRequestError(operationCreateSecretRotation, err)
	}

	if response.IsError() {
		return SecretRotation{}, errors.NewAPIErrorWithResponse(operationCreateSecretRotation, response, nil)
	}

	return body.SecretRotation, nil
}

func (client Client) GetSecretRotationById(request GetSecretRotationByIdRequest) (SecretRotation, error) {
	var body GetSecretRotationByIdResponse
	response, err := client.Config.HttpClient.
		R().
		SetDebug(true).
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Get("api/v2/secret-rotations/" + string(request.Provider) + "/" + request.ID)

	if err != nil {
		return SecretRotation{}, errors.NewGenericRequestError(operationGetSecretRotationById, err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return SecretRotation{}, ErrNotFound
	}

	if response.IsError() {
		return SecretRotation{}, errors.NewAPIErrorWithResponse(operationGetSecretRotationById, response, nil)
	}

	return body.SecretRotation, nil
}

func (client Client) UpdateSecretRotation(request UpdateSecretRotationRequest) (SecretRotation, error) {
	var body UpdateSecretRotationResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch("api/v2/secret-rotations/" + string(request.Provider) + "/" + request.ID)

	if err != nil {
		return SecretRotation{}, errors.NewGenericRequestError(operationUpdateSecretRotation, err)
	}

	if response.IsError() {
		return SecretRotation{}, errors.NewAPIErrorWithResponse(operationUpdateSecretRotation, response, nil)
	}

	return body.SecretRotation, nil
}

func (client Client) DeleteSecretRotation(request DeleteSecretRotationRequest) (SecretRotation, error) {
	var body DeleteSecretRotationResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetBody(request).
		SetHeader("User-Agent", USER_AGENT).
		Delete("api/v2/secret-rotations/" + string(request.Provider) + "/" + request.ID)

	if err != nil {
		return SecretRotation{}, errors.NewGenericRequestError(operationDeleteSecretRotation, err)
	}

	if response.IsError() {
		return SecretRotation{}, errors.NewAPIErrorWithResponse(operationDeleteSecretRotation, response, nil)
	}

	return body.SecretRotation, nil
}
