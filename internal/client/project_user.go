package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationInviteUsersToProject     = "CallInviteUsersToProject"
	operationDeleteProjectUser        = "CallDeleteProjectUser"
	operationUpdateProjectUser        = "CallUpdateProjectUser"
	operationGetProjectUserByUsername = "CallGetProjectUserByUsername"
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
		return nil, errors.NewGenericRequestError(operationInviteUsersToProject, err)
	}

	if response.IsError() {
		return nil, errors.NewAPIErrorWithResponse(operationInviteUsersToProject, response, nil)
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
		return DeleteProjectUserResponse{}, errors.NewGenericRequestError(operationDeleteProjectUser, err)
	}

	if response.IsError() {
		return DeleteProjectUserResponse{}, errors.NewAPIErrorWithResponse(operationDeleteProjectUser, response, nil)
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
		return UpdateProjectUserResponse{}, errors.NewGenericRequestError(operationUpdateProjectUser, err)
	}

	if response.IsError() {
		return UpdateProjectUserResponse{}, errors.NewAPIErrorWithResponse(operationUpdateProjectUser, response, nil)
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
		return GetProjectUserByUserNameResponse{}, errors.NewGenericRequestError(operationGetProjectUserByUsername, err)
	}

	if response.IsError() {

		if response.StatusCode() == http.StatusNotFound {
			return GetProjectUserByUserNameResponse{}, ErrNotFound
		}

		return GetProjectUserByUserNameResponse{}, errors.NewAPIErrorWithResponse(operationGetProjectUserByUsername, response, nil)
	}

	return projectUserResponse, nil
}
