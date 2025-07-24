package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationCreateProjectTemplate  = "CallCreateProjectTemplate"
	operationGetProjectTemplateById = "CallGetProjectTemplateById"
	operationUpdateProjectTemplate  = "CallUpdateProjectTemplate"
	operationDeleteProjectTemplate  = "CallDeleteProjectTemplate"
)

func (client Client) CreateProjectTemplate(request CreateProjectTemplateRequest) (ProjectTemplate, error) {
	var response CreateProjectTemplateResponse
	httpResponse, err := client.Config.HttpClient.
		R().
		SetResult(&response).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/project-templates")

	if err != nil {
		return ProjectTemplate{}, errors.NewGenericRequestError(operationCreateProjectTemplate, err)
	}

	if httpResponse.IsError() {
		return ProjectTemplate{}, errors.NewAPIErrorWithResponse(operationCreateProjectTemplate, httpResponse, nil)
	}

	return response.ProjectTemplate, nil
}

func (client Client) GetProjectTemplateById(id string) (ProjectTemplate, error) {
	var response GetProjectTemplateByIdResponse
	httpResponse, err := client.Config.HttpClient.
		R().
		SetResult(&response).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/project-templates/%s", id))

	if err != nil {
		return ProjectTemplate{}, errors.NewGenericRequestError(operationGetProjectTemplateById, err)
	}

	if httpResponse.StatusCode() == http.StatusNotFound {
		return ProjectTemplate{}, ErrNotFound
	}

	if httpResponse.IsError() {
		return ProjectTemplate{}, errors.NewAPIErrorWithResponse(operationGetProjectTemplateById, httpResponse, nil)
	}

	return response.ProjectTemplate, nil
}

func (client Client) UpdateProjectTemplate(request UpdateProjectTemplateRequest) (ProjectTemplate, error) {
	var response UpdateProjectTemplateResponse
	httpResponse, err := client.Config.HttpClient.
		R().
		SetResult(&response).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/project-templates/%s", request.ID))

	if err != nil {
		return ProjectTemplate{}, errors.NewGenericRequestError(operationUpdateProjectTemplate, err)
	}

	if httpResponse.IsError() {
		return ProjectTemplate{}, errors.NewAPIErrorWithResponse(operationUpdateProjectTemplate, httpResponse, nil)
	}

	return response.ProjectTemplate, nil
}

func (client Client) DeleteProjectTemplate(id string) (ProjectTemplate, error) {
	var response DeleteProjectTemplateResponse
	httpResponse, err := client.Config.HttpClient.
		R().
		SetResult(&response).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/project-templates/%s", id))

	if err != nil {
		return ProjectTemplate{}, errors.NewGenericRequestError(operationDeleteProjectTemplate, err)
	}

	if httpResponse.IsError() {
		if httpResponse.StatusCode() == http.StatusNotFound {
			return ProjectTemplate{}, ErrNotFound
		}
		return ProjectTemplate{}, errors.NewAPIErrorWithResponse(operationDeleteProjectTemplate, httpResponse, nil)
	}

	return response.ProjectTemplate, nil
}
