package infisicalclient

import (
	"fmt"
	"net/http"
	"net/url"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationCreateProjectScopedIdentity        = "CallCreateProjectScopedIdentity"
	operationGetProjectScopedIdentity           = "CallGetProjectScopedIdentity"
	operationUpdateProjectScopedIdentity        = "CallUpdateProjectScopedIdentity"
	operationDeleteProjectScopedIdentity        = "CallDeleteProjectScopedIdentity"
	operationUpdateProjectScopedIdentityRoles   = "CallUpdateProjectScopedIdentityRoles"
	operationGetProjectScopedIdentityMembership = "CallGetProjectScopedIdentityMembership"
)

// CreateProjectScopedIdentity creates a project-scoped machine identity.
func (client Client) CreateProjectScopedIdentity(request CreateProjectScopedIdentityRequest) (ProjectScopedIdentity, error) {
	var response CreateProjectScopedIdentityResponse
	httpResponse, err := client.Config.HttpClient.
		R().
		SetResult(&response).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(fmt.Sprintf("api/v1/projects/%s/identities", url.PathEscape(request.ProjectID)))

	if err != nil {
		return ProjectScopedIdentity{}, errors.NewGenericRequestError(operationCreateProjectScopedIdentity, err)
	}

	if httpResponse.IsError() {
		return ProjectScopedIdentity{}, errors.NewAPIErrorWithResponse(operationCreateProjectScopedIdentity, httpResponse, nil)
	}

	return response.Identity, nil
}

// GetProjectScopedIdentityByID fetches a project-scoped machine identity by its ID.
func (client Client) GetProjectScopedIdentityByID(request GetProjectScopedIdentityRequest) (ProjectScopedIdentity, error) {
	var response GetProjectScopedIdentityResponse
	httpResponse, err := client.Config.HttpClient.
		R().
		SetResult(&response).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/projects/%s/identities/%s", url.PathEscape(request.ProjectID), url.PathEscape(request.IdentityID)))

	if err != nil {
		return ProjectScopedIdentity{}, errors.NewGenericRequestError(operationGetProjectScopedIdentity, err)
	}

	if httpResponse.StatusCode() == http.StatusNotFound {
		return ProjectScopedIdentity{}, ErrNotFound
	}

	if httpResponse.IsError() {
		return ProjectScopedIdentity{}, errors.NewAPIErrorWithResponse(operationGetProjectScopedIdentity, httpResponse, nil)
	}

	return response.Identity, nil
}

// UpdateProjectScopedIdentity updates a project-scoped machine identity.
func (client Client) UpdateProjectScopedIdentity(request UpdateProjectScopedIdentityRequest) (ProjectScopedIdentity, error) {
	var response UpdateProjectScopedIdentityResponse
	httpResponse, err := client.Config.HttpClient.
		R().
		SetResult(&response).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/projects/%s/identities/%s", url.PathEscape(request.ProjectID), url.PathEscape(request.IdentityID)))

	if err != nil {
		return ProjectScopedIdentity{}, errors.NewGenericRequestError(operationUpdateProjectScopedIdentity, err)
	}

	if httpResponse.IsError() {
		return ProjectScopedIdentity{}, errors.NewAPIErrorWithResponse(operationUpdateProjectScopedIdentity, httpResponse, nil)
	}

	return response.Identity, nil
}

// DeleteProjectScopedIdentity deletes a project-scoped machine identity.
func (client Client) DeleteProjectScopedIdentity(request DeleteProjectScopedIdentityRequest) (ProjectScopedIdentity, error) {
	var response DeleteProjectScopedIdentityResponse
	httpResponse, err := client.Config.HttpClient.
		R().
		SetResult(&response).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/projects/%s/identities/%s", url.PathEscape(request.ProjectID), url.PathEscape(request.IdentityID)))

	if err != nil {
		return ProjectScopedIdentity{}, errors.NewGenericRequestError(operationDeleteProjectScopedIdentity, err)
	}

	if httpResponse.IsError() {
		if httpResponse.StatusCode() == http.StatusNotFound {
			return ProjectScopedIdentity{}, ErrNotFound
		}
		return ProjectScopedIdentity{}, errors.NewAPIErrorWithResponse(operationDeleteProjectScopedIdentity, httpResponse, nil)
	}

	return response.Identity, nil
}

// UpdateProjectScopedIdentityRoles sets the project roles on the membership of a project-scoped
// identity. A project-scoped identity always has an auto-created membership, so this updates it.
func (client Client) UpdateProjectScopedIdentityRoles(request UpdateProjectScopedIdentityRolesRequest) error {
	httpResponse, err := client.Config.HttpClient.
		R().
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/projects/%s/memberships/identities/%s", url.PathEscape(request.ProjectID), url.PathEscape(request.IdentityID)))

	if err != nil {
		return errors.NewGenericRequestError(operationUpdateProjectScopedIdentityRoles, err)
	}

	if httpResponse.IsError() {
		return errors.NewAPIErrorWithResponse(operationUpdateProjectScopedIdentityRoles, httpResponse, nil)
	}

	return nil
}

// GetProjectScopedIdentityMembership fetches the project membership (including roles) of a
// project-scoped identity.
func (client Client) GetProjectScopedIdentityMembership(request GetProjectIdentityByIDRequest) (GetProjectIdentityByIDResponse, error) {
	var response GetProjectIdentityByIDResponse
	httpResponse, err := client.Config.HttpClient.
		R().
		SetResult(&response).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/projects/%s/memberships/identities/%s", url.PathEscape(request.ProjectID), url.PathEscape(request.IdentityID)))

	if err != nil {
		return GetProjectIdentityByIDResponse{}, errors.NewGenericRequestError(operationGetProjectScopedIdentityMembership, err)
	}

	if httpResponse.StatusCode() == http.StatusNotFound {
		return GetProjectIdentityByIDResponse{}, ErrNotFound
	}

	if httpResponse.IsError() {
		return GetProjectIdentityByIDResponse{}, errors.NewAPIErrorWithResponse(operationGetProjectScopedIdentityMembership, httpResponse, nil)
	}

	return response, nil
}
