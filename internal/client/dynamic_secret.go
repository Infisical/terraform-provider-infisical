package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

type DynamicSecretProvider string

const (
	DynamicSecretProviderSQLDatabase DynamicSecretProvider = "sql-database"
	DynamicSecretProviderAWSIAM      DynamicSecretProvider = "aws-iam"
	DynamicSecretProviderKubernetes  DynamicSecretProvider = "kubernetes"
	DynamicSecretProviderMongoDb     DynamicSecretProvider = "mongo-db"
)

const (
	operationCreateDynamicSecret    = "CallCreateDynamicSecret"
	operationGetDynamicSecretByName = "CallGetDynamicSecretByName"
	operationUpdateDynamicSecret    = "CallUpdateDynamicSecret"
	operationDeleteDynamicSecret    = "CallDeleteDynamicSecret"
)

func (client Client) CreateDynamicSecret(request CreateDynamicSecretRequest) (DynamicSecret, error) {
	var body CreateDynamicSecretResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/dynamic-secrets")

	if err != nil {
		return DynamicSecret{}, errors.NewGenericRequestError(operationCreateDynamicSecret, err)
	}

	if response.IsError() {
		return DynamicSecret{}, errors.NewAPIErrorWithResponse(operationCreateDynamicSecret, response, nil)
	}

	return body.DynamicSecret, nil
}

func (client Client) GetDynamicSecretByName(request GetDynamicSecretByNameRequest) (DynamicSecret, error) {
	var body GetDynamicSecretByNameResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetQueryParam("projectSlug", request.ProjectSlug).
		SetQueryParam("environmentSlug", request.EnvironmentSlug).
		SetQueryParam("path", request.Path).
		Get(fmt.Sprintf("api/v1/dynamic-secrets/%s", request.Name))

	if err != nil {
		return DynamicSecret{}, errors.NewGenericRequestError(operationGetDynamicSecretByName, err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return DynamicSecret{}, ErrNotFound
	}

	if response.IsError() {
		return DynamicSecret{}, errors.NewAPIErrorWithResponse(operationGetDynamicSecretByName, response, nil)
	}

	return body.DynamicSecret, nil
}

func (client Client) UpdateDynamicSecret(request UpdateDynamicSecretRequest) (DynamicSecret, error) {
	var body UpdateDynamicSecretResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/dynamic-secrets/%s", request.Name))

	if err != nil {
		return DynamicSecret{}, errors.NewGenericRequestError(operationUpdateDynamicSecret, err)
	}

	if response.IsError() {
		return DynamicSecret{}, errors.NewAPIErrorWithResponse(operationUpdateDynamicSecret, response, nil)
	}

	return body.DynamicSecret, nil
}

func (client Client) DeleteDynamicSecret(request DeleteDynamicSecretRequest) (DynamicSecret, error) {
	var body DeleteDynamicSecretResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetBody(request).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/dynamic-secrets/%s", request.Name))

	if err != nil {
		return DynamicSecret{}, errors.NewGenericRequestError(operationDeleteDynamicSecret, err)
	}

	if response.IsError() {
		return DynamicSecret{}, errors.NewAPIErrorWithResponse(operationDeleteDynamicSecret, response, nil)
	}

	return body.DynamicSecret, nil
}
