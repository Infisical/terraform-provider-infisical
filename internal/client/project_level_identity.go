package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationCreateProjectLevelIdentity = "CallCreateProjectLevelIdentity"
	operationGetProjectLevelIdentity    = "CallGetProjectLevelIdentity"
	operationUpdateProjectLevelIdentity = "CallUpdateProjectLevelIdentity"
	operationDeleteProjectLevelIdentity = "CallDeleteProjectLevelIdentity"
)

// CreateProjectLevelIdentity creates a project-scoped machine identity.
func (client Client) CreateProjectLevelIdentity(request CreateProjectLevelIdentityRequest) (ProjectLevelIdentity, error) {
	var response CreateProjectLevelIdentityResponse
	httpResponse, err := client.Config.HttpClient.
		R().
		SetResult(&response).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(fmt.Sprintf("api/v1/projects/%s/identities", request.ProjectID))

	if err != nil {
		return ProjectLevelIdentity{}, errors.NewGenericRequestError(operationCreateProjectLevelIdentity, err)
	}

	if httpResponse.IsError() {
		return ProjectLevelIdentity{}, errors.NewAPIErrorWithResponse(operationCreateProjectLevelIdentity, httpResponse, nil)
	}

	return response.Identity, nil
}

// GetProjectLevelIdentityByID fetches a project-scoped machine identity by its ID.
func (client Client) GetProjectLevelIdentityByID(request GetProjectLevelIdentityRequest) (ProjectLevelIdentity, error) {
	var response GetProjectLevelIdentityResponse
	httpResponse, err := client.Config.HttpClient.
		R().
		SetResult(&response).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/projects/%s/identities/%s", request.ProjectID, request.IdentityID))

	if err != nil {
		return ProjectLevelIdentity{}, errors.NewGenericRequestError(operationGetProjectLevelIdentity, err)
	}

	if httpResponse.StatusCode() == http.StatusNotFound {
		return ProjectLevelIdentity{}, ErrNotFound
	}

	if httpResponse.IsError() {
		return ProjectLevelIdentity{}, errors.NewAPIErrorWithResponse(operationGetProjectLevelIdentity, httpResponse, nil)
	}

	return response.Identity, nil
}

// UpdateProjectLevelIdentity updates a project-scoped machine identity.
func (client Client) UpdateProjectLevelIdentity(request UpdateProjectLevelIdentityRequest) (ProjectLevelIdentity, error) {
	var response UpdateProjectLevelIdentityResponse
	httpResponse, err := client.Config.HttpClient.
		R().
		SetResult(&response).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/projects/%s/identities/%s", request.ProjectID, request.IdentityID))

	if err != nil {
		return ProjectLevelIdentity{}, errors.NewGenericRequestError(operationUpdateProjectLevelIdentity, err)
	}

	if httpResponse.IsError() {
		return ProjectLevelIdentity{}, errors.NewAPIErrorWithResponse(operationUpdateProjectLevelIdentity, httpResponse, nil)
	}

	return response.Identity, nil
}

// DeleteProjectLevelIdentity deletes a project-scoped machine identity.
func (client Client) DeleteProjectLevelIdentity(request DeleteProjectLevelIdentityRequest) (ProjectLevelIdentity, error) {
	var response DeleteProjectLevelIdentityResponse
	httpResponse, err := client.Config.HttpClient.
		R().
		SetResult(&response).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/projects/%s/identities/%s", request.ProjectID, request.IdentityID))

	if err != nil {
		return ProjectLevelIdentity{}, errors.NewGenericRequestError(operationDeleteProjectLevelIdentity, err)
	}

	if httpResponse.IsError() {
		if httpResponse.StatusCode() == http.StatusNotFound {
			return ProjectLevelIdentity{}, ErrNotFound
		}
		return ProjectLevelIdentity{}, errors.NewAPIErrorWithResponse(operationDeleteProjectLevelIdentity, httpResponse, nil)
	}

	return response.Identity, nil
}
