package infisicalclient

import (
	"fmt"
	"net/http"
)

func (client Client) CreateProject(request CreateProjectRequest) (CreateProjectResponse, error) {

	if request.Slug == "" {
		request = CreateProjectRequest{
			ProjectName:      request.ProjectName,
			OrganizationSlug: request.OrganizationSlug,
		}
	}

	var projectResponse CreateProjectResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&projectResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v2/workspace")

	if err != nil {
		return CreateProjectResponse{}, fmt.Errorf("CallCreateProject: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return CreateProjectResponse{}, fmt.Errorf("CallCreateProject: Unsuccessful response. [response=%s]", response)
	}

	return projectResponse, nil
}

func (client Client) DeleteProject(request DeleteProjectRequest) error {
	var projectResponse DeleteProjectResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&projectResponse).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v2/workspace/%s", request.Slug))

	if err != nil {
		return fmt.Errorf("CallDeleteProject: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return fmt.Errorf("CallDeleteProject: Unsuccessful response. [response=%s]", response)
	}

	return nil
}

func (client Client) GetProject(request GetProjectRequest) (ProjectWithEnvironments, error) {
	var projectResponse ProjectWithEnvironments
	response, err := client.Config.HttpClient.
		R().
		SetResult(&projectResponse).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v2/workspace/%s", request.Slug))

	if err != nil {
		return ProjectWithEnvironments{}, fmt.Errorf("CallGetProject: Unable to complete api request [err=%s]", err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return ProjectWithEnvironments{}, ErrNotFound
	}

	if response.IsError() {
		return ProjectWithEnvironments{}, fmt.Errorf("CallGetProject: Unsuccessful response. [response=%s]", response)
	}

	return projectResponse, nil
}

func (client Client) UpdateProject(request UpdateProjectRequest) (UpdateProjectResponse, error) {
	var projectResponse UpdateProjectResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&projectResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v2/workspace/%s", request.Slug))

	if err != nil {
		return UpdateProjectResponse{}, fmt.Errorf("CallUpdateProject: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return UpdateProjectResponse{}, fmt.Errorf("CallUpdateProject: Unsuccessful response. [response=%s]", response)
	}

	return projectResponse, nil
}

func (client Client) UpdateProjectAuditLogRetention(request UpdateProjectAuditLogRetentionRequest) (UpdateProjectAuditLogRetentionResponse, error) {
	var projectResponse UpdateProjectAuditLogRetentionResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&projectResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Put(fmt.Sprintf("api/v1/workspace/%s/audit-logs-retention", request.ProjectSlug))

	if err != nil {
		return UpdateProjectAuditLogRetentionResponse{}, fmt.Errorf("CallUpdateProjectAuditLogRetention: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return UpdateProjectAuditLogRetentionResponse{}, fmt.Errorf("CallUpdateProjectAuditLogRetention: Unsuccessful response. [response=%s]", response)
	}

	return projectResponse, nil
}

func (client Client) GetProjectById(request GetProjectByIdRequest) (ProjectWithEnvironments, error) {
	var projectResponse GetProjectByIdResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&projectResponse).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/workspace/%s", request.ID))

	if err != nil {
		return ProjectWithEnvironments{}, fmt.Errorf("CallGetProjectById: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return ProjectWithEnvironments{}, ErrNotFound
		}
		return ProjectWithEnvironments{}, fmt.Errorf("CallGetProjectById: Unsuccessful response. [response=%s]", response)
	}

	return projectResponse.Workspace, nil
}
