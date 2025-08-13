package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

type AppConnectionGcpMethod string

const (
	AppConnectionGcpMethodServiceAccountImpersonation AppConnectionGcpMethod = "service-account-impersonation"
)

type AppConnectionApp string

const (
	AppConnectionAppAWS                AppConnectionApp = "aws"
	AppConnectionAppGCP                AppConnectionApp = "gcp"
	AppConnectionAppAzure              AppConnectionApp = "azure"
	AppConnectionAppGithub             AppConnectionApp = "github"
	AppConnectionAppMySql              AppConnectionApp = "mysql"
	AppConnectionAppMsSql              AppConnectionApp = "mssql"
	AppConnectionAppPostgres           AppConnectionApp = "postgres"
	AppConnectionAppOracle             AppConnectionApp = "oracledb"
	AppConnectionApp1Password          AppConnectionApp = "1password"
	AppConnectionAppRender             AppConnectionApp = "render"
	AppConnectionAppAzureClientSecrets AppConnectionApp = "azure-client-secrets"
)

const (
	operationCreateAppConnection  = "CallCreateAppConnection"
	operationGetAppConnectionById = "CallGetAppConnectionById"
	operationUpdateAppConnection  = "CallUpdateAppConnection"
	operationDeleteAppConnection  = "CallDeleteAppConnection"
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
		return AppConnection{}, errors.NewGenericRequestError(operationCreateAppConnection, err)
	}

	if response.IsError() {
		return AppConnection{}, errors.NewAPIErrorWithResponse(operationCreateAppConnection, response, nil)
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
		return AppConnection{}, errors.NewGenericRequestError(operationGetAppConnectionById, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return AppConnection{}, ErrNotFound
		}
		return AppConnection{}, errors.NewAPIErrorWithResponse(operationGetAppConnectionById, response, nil)
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
		return AppConnection{}, errors.NewGenericRequestError(operationUpdateAppConnection, err)
	}

	if response.IsError() {
		return AppConnection{}, errors.NewAPIErrorWithResponse(operationUpdateAppConnection, response, nil)
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
		return AppConnection{}, errors.NewGenericRequestError(operationDeleteAppConnection, err)
	}

	if response.IsError() {
		return AppConnection{}, errors.NewAPIErrorWithResponse(operationDeleteAppConnection, response, nil)
	}

	return body.AppConnection, nil
}
