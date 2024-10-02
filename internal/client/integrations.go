package infisicalclient

import (
	"fmt"
)

func (client Client) CallCreateIntegration(request CreateIntegrationRequest) (CreateIntegrationResponse, error) {
	var body CreateIntegrationResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/integration")

	if err != nil {
		return CreateIntegrationResponse{}, fmt.Errorf("CallCreateIntegration: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return CreateIntegrationResponse{}, fmt.Errorf("CallCreateIntegration: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body, nil
}

func (client Client) CallGetIntegration(request GetIntegrationRequest) (GetIntegrationResponse, error) {
	var body GetIntegrationResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/integration/%s", request.ID))

	if err != nil {
		return GetIntegrationResponse{}, fmt.Errorf("CallGetIntegration: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return GetIntegrationResponse{}, fmt.Errorf("CallGetIntegration: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body, nil
}

func (client Client) CallUpdateIntegration(request UpdateIntegrationRequest) (UpdateIntegrationResponse, error) {
	var body UpdateIntegrationResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/integration/%s", request.ID))

	if err != nil {
		return UpdateIntegrationResponse{}, fmt.Errorf("CallUpdateIntegration: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return UpdateIntegrationResponse{}, fmt.Errorf("CallUpdateIntegration: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body, nil
}
