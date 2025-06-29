package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationCreateIntegration = "CallCreateIntegration"
	operationGetIntegration    = "CallGetIntegration"
	operationUpdateIntegration = "CallUpdateIntegration"
)

func (client Client) CreateIntegration(request CreateIntegrationRequest) (CreateIntegrationResponse, error) {
	var body CreateIntegrationResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/integration")

	if err != nil {
		return CreateIntegrationResponse{}, errors.NewGenericRequestError(operationCreateIntegration, err)
	}

	if response.IsError() {
		return CreateIntegrationResponse{}, errors.NewAPIErrorWithResponse(operationCreateIntegration, response, nil)
	}

	return body, nil
}

func (client Client) GetIntegration(request GetIntegrationRequest) (GetIntegrationResponse, error) {
	var body GetIntegrationResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/integration/%s", request.ID))

	if err != nil {
		return GetIntegrationResponse{}, errors.NewGenericRequestError(operationGetIntegration, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return GetIntegrationResponse{}, ErrNotFound
		}
		return GetIntegrationResponse{}, errors.NewAPIErrorWithResponse(operationGetIntegration, response, nil)
	}

	return body, nil
}

func (client Client) UpdateIntegration(request UpdateIntegrationRequest) (UpdateIntegrationResponse, error) {
	var body UpdateIntegrationResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/integration/%s", request.ID))

	if err != nil {
		return UpdateIntegrationResponse{}, errors.NewGenericRequestError(operationUpdateIntegration, err)
	}

	if response.IsError() {
		return UpdateIntegrationResponse{}, errors.NewAPIErrorWithResponse(operationUpdateIntegration, response, nil)
	}

	return body, nil
}
