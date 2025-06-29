package infisicalclient

import (
	"fmt"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationCreateProjectRole      = "CallCreateProjectRole"
	operationDeleteProjectRole      = "CallDeleteProjectRole"
	operationUpdateProjectRole      = "CallUpdateProjectRole"
	operationGetProjectRoleBySlug   = "CallGetProjectRoleBySlug"
	operationCreateProjectRoleV2    = "CallCreateProjectRoleV2"
	operationUpdateProjectRoleV2    = "CallUpdateProjectRoleV2"
	operationGetProjectRoleBySlugV2 = "CallGetProjectRoleBySlugV2"
)

func (client Client) CreateProjectRole(request CreateProjectRoleRequest) (CreateProjectRoleResponse, error) {
	var responeData CreateProjectRoleResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responeData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(fmt.Sprintf("api/v1/workspace/%s/roles", request.ProjectSlug))

	if err != nil {
		return CreateProjectRoleResponse{}, errors.NewGenericRequestError(operationCreateProjectRole, err)
	}

	if response.IsError() {
		return CreateProjectRoleResponse{}, errors.NewAPIErrorWithResponse(operationCreateProjectRole, response, nil)
	}

	return responeData, nil
}

func (client Client) CreateProjectRoleV2(request CreateProjectRoleV2Request) (CreateProjectRoleV2Response, error) {
	var responseData CreateProjectRoleV2Response
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(fmt.Sprintf("api/v2/workspace/%s/roles", request.ProjectId))

	if err != nil {
		return CreateProjectRoleV2Response{}, errors.NewGenericRequestError(operationCreateProjectRoleV2, err)
	}

	if response.IsError() {
		return CreateProjectRoleV2Response{}, errors.NewAPIErrorWithResponse(operationCreateProjectRoleV2, response, nil)
	}

	return responseData, nil
}

func (client Client) DeleteProjectRole(request DeleteProjectRoleRequest) (DeleteProjectRoleResponse, error) {
	var responseData DeleteProjectRoleResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Delete(fmt.Sprintf("/api/v1/workspace/%s/roles/%s", request.ProjectSlug, request.RoleId))

	if err != nil {
		return DeleteProjectRoleResponse{}, errors.NewGenericRequestError(operationDeleteProjectRole, err)
	}

	if response.IsError() {
		return DeleteProjectRoleResponse{}, errors.NewAPIErrorWithResponse(operationDeleteProjectRole, response, nil)
	}

	return responseData, nil
}

func (client Client) UpdateProjectRole(request UpdateProjectRoleRequest) (UpdateProjectRoleResponse, error) {
	var responseData UpdateProjectRoleResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/workspace/%s/roles/%s", request.ProjectSlug, request.RoleId))

	if err != nil {
		return UpdateProjectRoleResponse{}, errors.NewGenericRequestError(operationUpdateProjectRole, err)
	}

	if response.IsError() {
		return UpdateProjectRoleResponse{}, errors.NewAPIErrorWithResponse(operationUpdateProjectRole, response, nil)
	}

	return responseData, nil
}

func (client Client) UpdateProjectRoleV2(request UpdateProjectRoleV2Request) (UpdateProjectRoleV2Response, error) {
	var responseData UpdateProjectRoleV2Response
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v2/workspace/%s/roles/%s", request.ProjectId, request.RoleId))

	if err != nil {
		return UpdateProjectRoleV2Response{}, errors.NewGenericRequestError(operationUpdateProjectRoleV2, err)
	}

	if response.IsError() {
		return UpdateProjectRoleV2Response{}, errors.NewAPIErrorWithResponse(operationUpdateProjectRoleV2, response, nil)
	}

	return responseData, nil
}

func (client Client) GetProjectRoleBySlug(request GetProjectRoleBySlugRequest) (GetProjectRoleBySlugResponse, error) {
	var responseData GetProjectRoleBySlugResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Get(fmt.Sprintf("api/v1/workspace/%s/roles/slug/%s", request.ProjectSlug, request.RoleSlug))

	if err != nil {
		return GetProjectRoleBySlugResponse{}, errors.NewGenericRequestError(operationGetProjectRoleBySlug, err)
	}

	if response.IsError() {
		return GetProjectRoleBySlugResponse{}, errors.NewAPIErrorWithResponse(operationGetProjectRoleBySlug, response, nil)
	}

	return responseData, nil
}

func (client Client) GetProjectRoleBySlugV2(request GetProjectRoleBySlugV2Request) (GetProjectRoleBySlugV2Response, error) {
	var responseData GetProjectRoleBySlugV2Response
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Get(fmt.Sprintf("api/v2/workspace/%s/roles/slug/%s", request.ProjectId, request.RoleSlug))

	if err != nil {
		return GetProjectRoleBySlugV2Response{}, errors.NewGenericRequestError(operationGetProjectRoleBySlugV2, err)
	}

	if response.IsError() {
		return GetProjectRoleBySlugV2Response{}, errors.NewAPIErrorWithResponse(operationGetProjectRoleBySlugV2, response, nil)
	}

	return responseData, nil
}
