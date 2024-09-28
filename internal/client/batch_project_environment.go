package infisicalclient

import (
	"fmt"
	"net/http"
	"strings"
)

func (client Client) CreateBatchProjectEnvironments(request CreateBatchProjectEnvironmentsRequest) (CreateBatchProjectEnvironmentsResponse, error) {
	var body CreateBatchProjectEnvironmentsResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(fmt.Sprintf("api/v1/workspace/%s/batch/environments", request.ProjectID))

	if err != nil {
		return CreateBatchProjectEnvironmentsResponse{}, fmt.Errorf("CreateBatchProjectEnvironment: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return CreateBatchProjectEnvironmentsResponse{}, fmt.Errorf("CreateBatchProjectEnvironment: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body, nil
}

func (client Client) UpdateBatchProjectEnvironments(request UpdateBatchProjectEnvironmentsRequest) (UpdateBatchProjectEnvironmentsResponse, error) {
	var body UpdateBatchProjectEnvironmentsResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(map[string]interface{}{
			"environments": request.Environments,
		}).
		Patch(fmt.Sprintf("api/v1/workspace/%s/batch/environments", request.ProjectID))

	if err != nil {
		return UpdateBatchProjectEnvironmentsResponse{}, fmt.Errorf("UpdateBatchProjectEnvironment: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return UpdateBatchProjectEnvironmentsResponse{}, fmt.Errorf("UpdateBatchProjectEnvironment: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body, nil
}

func (client Client) DeleteBatchProjectEnvironments(request DeleteBatchProjectEnvironmentsRequest) (DeleteBatchProjectEnvironmentsResponse, error) {
	var body DeleteBatchProjectEnvironmentsResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Delete(fmt.Sprintf("api/v1/workspace/%s/batch/environments", request.ProjectID))

	if err != nil {
		return DeleteBatchProjectEnvironmentsResponse{}, fmt.Errorf("DeleteBatchProjectEnvironment: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return DeleteBatchProjectEnvironmentsResponse{}, fmt.Errorf("DeleteBatchProjectEnvironment: Unsuccessful response. [response=%s]", string(response.Body()))
	}

	return body, nil
}

func (client Client) GetBatchProjectEnvironments(request GetBatchProjectEnvironmentsRequest) (GetBatchProjectEnvironmentsResponse, error) {
	var body GetBatchProjectEnvironmentsResponse

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT)

	response, err := httpRequest.Get(fmt.Sprintf("api/v1/workspace/%s/batch/environments?environmentIds=%s",
		request.ProjectID,
		strings.Join(request.EnvironmentIds, ","),
	))

	if response.StatusCode() == http.StatusNotFound {
		return GetBatchProjectEnvironmentsResponse{}, ErrNotFound
	}

	if err != nil {
		return GetBatchProjectEnvironmentsResponse{}, fmt.Errorf("GetBatchProjectEnvironments: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return GetBatchProjectEnvironmentsResponse{}, fmt.Errorf("GetBatchProjectEnvironments: Unsuccessful response. [response=%v]", string(response.Body()))
	}

	return body, nil
}
