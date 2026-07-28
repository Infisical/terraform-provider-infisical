package infisicalclient

import (
	"fmt"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationCreateProxiedService    = "CallCreateProxiedService"
	operationGetProxiedService       = "CallGetProxiedService"
	operationGetProxiedServiceByName = "CallGetProxiedServiceByName"
	operationUpdateProxiedService    = "CallUpdateProxiedService"
	operationDeleteProxiedService    = "CallDeleteProxiedService"
)

func (client Client) CreateProxiedService(request CreateProxiedServiceRequest) (CreateProxiedServiceResponse, error) {
	var serviceResponse CreateProxiedServiceResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&serviceResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/proxied-services")

	if err != nil {
		return CreateProxiedServiceResponse{}, errors.NewGenericRequestError(operationCreateProxiedService, err)
	}

	if response.IsError() {
		return CreateProxiedServiceResponse{}, errors.NewAPIErrorWithResponse(operationCreateProxiedService, response, nil)
	}

	return serviceResponse, nil
}

func (client Client) GetProxiedService(request GetProxiedServiceRequest) (GetProxiedServiceResponse, error) {
	var serviceResponse GetProxiedServiceResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&serviceResponse).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/proxied-services/%s", request.ServiceId))

	if err != nil {
		return GetProxiedServiceResponse{}, errors.NewGenericRequestError(operationGetProxiedService, err)
	}

	if response.IsError() {
		if response.StatusCode() == 404 || response.StatusCode() == 422 {
			return GetProxiedServiceResponse{}, ErrNotFound
		}
		return GetProxiedServiceResponse{}, errors.NewAPIErrorWithResponse(operationGetProxiedService, response, nil)
	}

	return serviceResponse, nil
}

// GetProxiedServiceByName resolves a service from its folder scope. The get-by-id response carries
// no project, environment, or secret path, so importing needs this lookup to reconstruct them.
func (client Client) GetProxiedServiceByName(request GetProxiedServiceByNameRequest) (GetProxiedServiceResponse, error) {
	var serviceResponse GetProxiedServiceResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&serviceResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetQueryParams(map[string]string{
			"projectId":   request.ProjectId,
			"environment": request.Environment,
			"secretPath":  request.SecretPath,
		}).
		Get(fmt.Sprintf("api/v1/proxied-services/slug/%s", request.Name))

	if err != nil {
		return GetProxiedServiceResponse{}, errors.NewGenericRequestError(operationGetProxiedServiceByName, err)
	}

	if response.IsError() {
		if response.StatusCode() == 404 || response.StatusCode() == 422 {
			return GetProxiedServiceResponse{}, ErrNotFound
		}
		return GetProxiedServiceResponse{}, errors.NewAPIErrorWithResponse(operationGetProxiedServiceByName, response, nil)
	}

	return serviceResponse, nil
}

func (client Client) UpdateProxiedService(request UpdateProxiedServiceRequest) (UpdateProxiedServiceResponse, error) {
	var serviceResponse UpdateProxiedServiceResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&serviceResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/proxied-services/%s", request.ServiceId))

	if err != nil {
		return UpdateProxiedServiceResponse{}, errors.NewGenericRequestError(operationUpdateProxiedService, err)
	}

	if response.IsError() {
		return UpdateProxiedServiceResponse{}, errors.NewAPIErrorWithResponse(operationUpdateProxiedService, response, nil)
	}

	return serviceResponse, nil
}

func (client Client) DeleteProxiedService(request DeleteProxiedServiceRequest) (DeleteProxiedServiceResponse, error) {
	var serviceResponse DeleteProxiedServiceResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&serviceResponse).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/proxied-services/%s", request.ServiceId))

	if err != nil {
		return DeleteProxiedServiceResponse{}, errors.NewGenericRequestError(operationDeleteProxiedService, err)
	}

	if response.IsError() {
		return DeleteProxiedServiceResponse{}, errors.NewAPIErrorWithResponse(operationDeleteProxiedService, response, nil)
	}

	return serviceResponse, nil
}
