package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationCreateProjectEnvironment  = "CallCreateProjectEnvironment"
	operationDeleteProjectEnvironment  = "CallDeleteProjectEnvironment"
	operationGetProjectEnvironmentByID = "CallGetProjectEnvironmentByID"
	operationUpdateProjectEnvironment  = "CallUpdateProjectEnvironment"
)

func (client Client) CreateProjectEnvironment(request CreateProjectEnvironmentRequest) (CreateProjectEnvironmentResponse, error) {
	var body CreateProjectEnvironmentResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(fmt.Sprintf("api/v1/workspace/%s/environments", request.ProjectID))

	if err != nil {
		return CreateProjectEnvironmentResponse{}, errors.NewGenericRequestError(operationCreateProjectEnvironment, err)
	}

	if response.IsError() {
		return CreateProjectEnvironmentResponse{}, errors.NewAPIErrorWithResponse(operationCreateProjectEnvironment, response, nil)
	}

	return body, nil
}

func (client Client) DeleteProjectEnvironment(request DeleteProjectEnvironmentRequest) (DeleteProjectEnvironmentResponse, error) {
	var body DeleteProjectEnvironmentResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Delete(fmt.Sprintf("api/v1/workspace/%s/environments/%s", request.ProjectID, request.ID))

	if err != nil {
		return DeleteProjectEnvironmentResponse{}, errors.NewGenericRequestError(operationDeleteProjectEnvironment, err)
	}

	if response.IsError() {
		return DeleteProjectEnvironmentResponse{}, errors.NewAPIErrorWithResponse(operationDeleteProjectEnvironment, response, nil)
	}

	return body, nil
}

func (client Client) GetProjectEnvironmentByID(request GetProjectEnvironmentByIDRequest) (GetProjectEnvironmentByIDResponse, error) {
	var body GetProjectEnvironmentByIDResponse

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT)

	response, err := httpRequest.Get(fmt.Sprintf("api/v1/workspace/environments/%s", request.ID))

	if response.StatusCode() == http.StatusNotFound {
		return GetProjectEnvironmentByIDResponse{}, ErrNotFound
	}

	if err != nil {
		return GetProjectEnvironmentByIDResponse{}, errors.NewGenericRequestError(operationGetProjectEnvironmentByID, err)
	}

	if response.IsError() {
		return GetProjectEnvironmentByIDResponse{}, errors.NewAPIErrorWithResponse(operationGetProjectEnvironmentByID, response, nil)
	}

	return body, nil
}

func (client Client) UpdateProjectEnvironment(request UpdateProjectEnvironmentRequest) (UpdateProjectEnvironmentResponse, error) {
	var body UpdateProjectEnvironmentResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/workspace/%s/environments/%s", request.ProjectID, request.ID))

	if err != nil {
		return UpdateProjectEnvironmentResponse{}, errors.NewGenericRequestError(operationUpdateProjectEnvironment, err)
	}

	if response.IsError() {
		return UpdateProjectEnvironmentResponse{}, errors.NewAPIErrorWithResponse(operationUpdateProjectEnvironment, response, nil)
	}

	return body, nil
}
