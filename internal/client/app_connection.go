package infisicalclient

import (
	"fmt"
	"net/http"
)

type AppConnectionGcpMethod string

const (
	AppConnectionGcpMethodServiceAccountImpersonation AppConnectionGcpMethod = "service-account-impersonation"
)

type AppConnectionApp string

const (
	AppConnectionAppGCP   AppConnectionApp = "gcp"
	AppConnectionAppAzure AppConnectionApp = "azure"
)

func (client Client) CreateAppConnection(request CreateAppConnectionRequest) (AppConnection, error) {
	var body CreateAppConnectionResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/app-connections/" + string(request.App))

	if err != nil {
		return AppConnection{}, fmt.Errorf("CreateAppConnection: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return AppConnection{}, fmt.Errorf("CreateAppConnection: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body.AppConnection, nil
}

func (client Client) GetAppConnectionById(request GetAppConnectionByIdRequest) (AppConnection, error) {
	var body GetAppConnectionByIdResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/app-connections/%s/%s", request.App, request.ID))

	if err != nil {
		return AppConnection{}, fmt.Errorf("GetAppConnectionById: Unable to complete api request [err=%s]", err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return AppConnection{}, ErrNotFound
	}

	if response.IsError() {
		return AppConnection{}, fmt.Errorf("GetAppConnectionById: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body.AppConnection, nil
}

func (client Client) UpdateAppConnection(request UpdateAppConnectionRequest) (AppConnection, error) {
	var body UpdateAppConnectionResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/app-connections/%s/%s", request.App, request.ID))

	if err != nil {
		return AppConnection{}, fmt.Errorf("UpdateAppConnection: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return AppConnection{}, fmt.Errorf("UpdateAppConnection: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body.AppConnection, nil
}

func (client Client) DeleteAppConnection(request DeleteAppConnectionRequest) (AppConnection, error) {
	var body DeleteAppConnectionResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/app-connections/%s/%s", request.App, request.ID))

	if err != nil {
		return AppConnection{}, fmt.Errorf("DeleteAppConnection: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return AppConnection{}, fmt.Errorf("DeleteAppConnection: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body.AppConnection, nil
}
