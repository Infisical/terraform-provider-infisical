package infisicalclient

import (
	"fmt"
	"net/http"
)

func (client Client) CreateProjectGroup(request CreateProjectGroupRequest) (CreateProjectGroupResponse, error) {
	var responseData CreateProjectGroupResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&responseData).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(fmt.Sprintf("api/v2/workspace/%s/groups/%s", request.ProjectId, request.GroupId))

	if err != nil {
		return CreateProjectGroupResponse{}, fmt.Errorf("CallCreateProjectGroup: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return CreateProjectGroupResponse{}, fmt.Errorf("CallCreateProjectGroup: Unsuccessful response. [response=%s]", response)
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
		return GetProjectGroupMembershipResponse{}, fmt.Errorf("GetProjectGroupMembershipResponse: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return GetProjectGroupMembershipResponse{}, fmt.Errorf("GetProjectGroupMembershipResponse: Unsuccessful response. [response=%s]", response)
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
		return UpdateProjectGroupResponse{}, fmt.Errorf("CallUpdateProjectGroup: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return UpdateProjectGroupResponse{}, fmt.Errorf("CallUpdateProjectGroup: Unsuccessful response. [response=%s]", response)
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
		return DeleteProjectGroupResponse{}, fmt.Errorf("CallDeleteProjectGroup: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return DeleteProjectGroupResponse{}, fmt.Errorf("CallDeleteProjectGroup: Unsuccessful response. [response=%s]", response)
	}

	return responseData, nil
}
