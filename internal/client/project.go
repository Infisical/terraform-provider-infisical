package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationCreateProject                  = "CallCreateProject"
	operationDeleteProject                  = "CallDeleteProject"
	operationGetProject                     = "CallGetProject"
	operationUpdateProject                  = "CallUpdateProject"
	operationUpdateProjectAuditLogRetention = "CallUpdateProjectAuditLogRetention"
	operationGetProjectById                 = "CallGetProjectById"
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
		return CreateProjectResponse{}, errors.NewGenericRequestError(operationCreateProject, err)
	}

	if response.IsError() {
		return CreateProjectResponse{}, errors.NewAPIErrorWithResponse(operationCreateProject, response, nil)
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
		return errors.NewGenericRequestError(operationDeleteProject, err)
	}

	if response.IsError() {
		return errors.NewAPIErrorWithResponse(operationDeleteProject, response, nil)
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
		return ProjectWithEnvironments{}, errors.NewGenericRequestError(operationGetProject, err)
	}

	if response.StatusCode() == http.StatusNotFound {
		return ProjectWithEnvironments{}, ErrNotFound
	}

	if response.IsError() {
		return ProjectWithEnvironments{}, errors.NewAPIErrorWithResponse(operationGetProject, response, nil)
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
		return UpdateProjectResponse{}, errors.NewGenericRequestError(operationUpdateProject, err)
	}

	if response.IsError() {
		return UpdateProjectResponse{}, errors.NewAPIErrorWithResponse(operationUpdateProject, response, nil)
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
		return UpdateProjectAuditLogRetentionResponse{}, errors.NewGenericRequestError(operationUpdateProjectAuditLogRetention, err)
	}

	if response.IsError() {
		return UpdateProjectAuditLogRetentionResponse{}, errors.NewAPIErrorWithResponse(operationUpdateProjectAuditLogRetention, response, nil)
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
		return ProjectWithEnvironments{}, errors.NewGenericRequestError(operationGetProjectById, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return ProjectWithEnvironments{}, ErrNotFound
		}
		return ProjectWithEnvironments{}, errors.NewAPIErrorWithResponse(operationGetProjectById, response, nil)
	}

	return projectResponse.Workspace, nil
}
