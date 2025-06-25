package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationCreateProjectGroup        = "CallCreateProjectGroup"
	operationGetProjectGroupMembership = "CallGetProjectGroupMembership"
	operationUpdateProjectGroup        = "CallUpdateProjectGroup"
	operationDeleteProjectGroup        = "CallDeleteProjectGroup"
)

func (client Client) CreateProjectGroup(request CreateProjectGroupRequest) (CreateProjectGroupResponse, error) {
	var responseData CreateProjectGroupResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(fmt.Sprintf("api/v2/workspace/%s/groups/%s", request.ProjectId, request.GroupIdOrName))

	if err != nil {
		return CreateProjectGroupResponse{}, errors.NewGenericRequestError(operationCreateProjectGroup, err)
	}

	if response.IsError() {
		return CreateProjectGroupResponse{}, errors.NewAPIErrorWithResponse(operationCreateProjectGroup, response, nil)
	}

	return responseData, nil
}

func (client Client) GetProjectGroupMembership(request GetProjectGroupMembershipRequest) (GetProjectGroupMembershipResponse, error) {
	var responseData GetProjectGroupMembershipResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Get(fmt.Sprintf("api/v2/workspace/%s/groups/%s", request.ProjectId, request.GroupId))

	if response.StatusCode() == http.StatusNotFound {
		return GetProjectGroupMembershipResponse{}, ErrNotFound
	}

	if err != nil {
		return GetProjectGroupMembershipResponse{}, errors.NewGenericRequestError(operationGetProjectGroupMembership, err)
	}

	if response.IsError() {
		return GetProjectGroupMembershipResponse{}, errors.NewAPIErrorWithResponse(operationGetProjectGroupMembership, response, nil)
	}

	return responseData, nil
}

func (client Client) UpdateProjectGroup(request UpdateProjectGroupRequest) (UpdateProjectGroupResponse, error) {
	var responseData UpdateProjectGroupResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v2/workspace/%s/groups/%s", request.ProjectId, request.GroupId))

	if err != nil {
		return UpdateProjectGroupResponse{}, errors.NewGenericRequestError(operationUpdateProjectGroup, err)
	}

	if response.IsError() {
		return UpdateProjectGroupResponse{}, errors.NewAPIErrorWithResponse(operationUpdateProjectGroup, response, nil)
	}

	return responseData, nil
}

func (client Client) DeleteProjectGroup(request DeleteProjectGroupRequest) (DeleteProjectGroupResponse, error) {
	var responseData DeleteProjectGroupResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Delete(fmt.Sprintf("/api/v2/workspace/%s/groups/%s", request.ProjectId, request.GroupId))

	if err != nil {
		return DeleteProjectGroupResponse{}, errors.NewGenericRequestError(operationDeleteProjectGroup, err)
	}

	if response.IsError() {
		return DeleteProjectGroupResponse{}, errors.NewAPIErrorWithResponse(operationDeleteProjectGroup, response, nil)
	}

	return responseData, nil
}
