package infisicalclient

import (
	"fmt"
	"net/http"
)

type DynamicSecretProvider string

const (
	DynamicSecretProviderSQLDatabase DynamicSecretProvider = "sql-database"
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
		return DynamicSecret{}, fmt.Errorf("CreateDynamicSecret: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return DynamicSecret{}, fmt.Errorf("CreateDynamicSecret: Unsuccessful response. [response=%s]", string(response.Body()))
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
		return DynamicSecret{}, fmt.Errorf("GetDynamicSecretByName: Unable to complete api request [err=%s]", err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return DynamicSecret{}, ErrNotFound
	}

	if response.IsError() {
		return DynamicSecret{}, fmt.Errorf("GetDynamicSecretByName: Unsuccessful response. [response=%s]", string(response.Body()))
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
		return DynamicSecret{}, fmt.Errorf("UpdateDynamicSecret: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return DynamicSecret{}, fmt.Errorf("UpdateDynamicSecret: Unsuccessful response. [response=%s]", string(response.Body()))
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
		return DynamicSecret{}, fmt.Errorf("DeleteDynamicSecret: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return DynamicSecret{}, fmt.Errorf("DeleteDynamicSecret: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body.DynamicSecret, nil
}
