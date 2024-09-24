package infisicalclient

import (
	"fmt"
	"net/http"
)

func (client Client) InviteUsersToProject(request InviteUsersToProjectRequest) ([]ProjectMemberships, error) {
	var inviteUsersToProjectResponse InviteUsersToProjectResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&inviteUsersToProjectResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(fmt.Sprintf("api/v2/workspace/%s/memberships", request.ProjectID))

	if err != nil {
		return nil, fmt.Errorf("CallInviteUsersToProject: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return nil, fmt.Errorf("InviteUsersToProjectRequest: Unsuccessful response. [response=%s]", response)
	}

	return inviteUsersToProjectResponse.Members, nil
}

func (client Client) DeleteProjectUser(request DeleteProjectUserRequest) (DeleteProjectUserResponse, error) {
	var projectUserResponse DeleteProjectUserResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&projectUserResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Delete(fmt.Sprintf("api/v2/workspace/%s/memberships", request.ProjectID))

	if err != nil {
		return DeleteProjectUserResponse{}, fmt.Errorf("CallDeleteProjectUser: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return DeleteProjectUserResponse{}, fmt.Errorf("CallDeleteProjectUser: Unsuccessful response. [response=%s]", response)
	}

	return projectUserResponse, nil
}

func (client Client) UpdateProjectUser(request UpdateProjectUserRequest) (UpdateProjectUserResponse, error) {
	var projectUserResponse UpdateProjectUserResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&projectUserResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/workspace/%s/memberships/%s", request.ProjectID, request.MembershipID))

	if err != nil {
		return UpdateProjectUserResponse{}, fmt.Errorf("UpdateProjectUserResponse: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {
		return UpdateProjectUserResponse{}, fmt.Errorf("UpdateProjectUserResponse: Unsuccessful response. [response=%s]", response)
	}

	return projectUserResponse, nil
}

func (client Client) GetProjectUserByUsername(request GetProjectUserByUserNameRequest) (GetProjectUserByUserNameResponse, error) {
	var projectUserResponse GetProjectUserByUserNameResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&projectUserResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(fmt.Sprintf("api/v1/workspace/%s/memberships/details", request.ProjectID))

	if err != nil {
		return GetProjectUserByUserNameResponse{}, fmt.Errorf("CallGetProjectUserByUsername: Unable to complete api request [err=%s]", err)
	}

	if response.IsError() {

		if response.StatusCode() == http.StatusNotFound {
			return GetProjectUserByUserNameResponse{}, ErrNotFound
		}

		return GetProjectUserByUserNameResponse{}, fmt.Errorf("CallGetProjectUserByUsername: Unsuccessful response. [response=%s]", response)
	}

	return projectUserResponse, nil
}
