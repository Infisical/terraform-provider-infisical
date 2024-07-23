package infisicalclient

import (
	"fmt"
	"net/http"
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
		return CreateProjectEnvironmentResponse{}, fmt.Errorf("CallCreateProjectEnvironment: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return CreateProjectEnvironmentResponse{}, fmt.Errorf("CallCreateProjectEnvironment: Unsuccessful response. [response=%s]", string(response.Body()))
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
		return DeleteProjectEnvironmentResponse{}, fmt.Errorf("CallDeleteProjectEnvironment: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return DeleteProjectEnvironmentResponse{}, fmt.Errorf("CallDeleteProjectEnvironment Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body, nil
}

func (client Client) GetProjectEnvironmentByID(request GetProjectEnvironmentByIDRequest) (GetProjectEnvironmentByIDResponse, error) {
	var body GetProjectEnvironmentByIDResponse

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT)

	response, err := httpRequest.Get(fmt.Sprintf("api/v1/workspace/%s/environments/%s", request.ProjectID, request.ID))

	if response.StatusCode() == http.StatusNotFound {
		return GetProjectEnvironmentByIDResponse{}, ErrNotFound
	}

	if err != nil {
		return GetProjectEnvironmentByIDResponse{}, fmt.Errorf("GetProjectEnvironmentByID: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return GetProjectEnvironmentByIDResponse{}, fmt.Errorf("GetProjectEnvironmentByID: Unsuccessful response. [response=%v]", string(response.Body()))
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
		return UpdateProjectEnvironmentResponse{}, fmt.Errorf("CallUpdateProjectEnvironment: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return UpdateProjectEnvironmentResponse{}, fmt.Errorf("CallUpdateProjectEnvironment: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body, nil
}
