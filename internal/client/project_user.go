package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationInviteUsersToProject         = "CallInviteUsersToProject"
	operationDeleteProjectUser            = "CallDeleteProjectUser"
	operationUpdateProjectUser            = "CallUpdateProjectUser"
	operationGetProjectUserByUsername     = "CallGetProjectUserByUsername"
	operationGetProjectMembershipByUserID = "CallGetProjectMembershipByUserID"
	operationGetProjectMemberships        = "CallGetProjectMemberships"
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

func (client Client) GetProjectMemberships(request GetProjectMembershipsRequest) (GetProjectMembershipsResponse, error) {
	var body GetProjectMembershipsResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/projects/%s/memberships", request.ProjectID))

	if err != nil {
		return GetProjectMembershipsResponse{}, errors.NewGenericRequestError(operationGetProjectMemberships, err)
	}

	if response.IsError() {
		return GetProjectMembershipsResponse{}, errors.NewAPIErrorWithResponse(operationGetProjectMemberships, response, nil)
	}

	return body, nil
}

func (client Client) GetProjectMembershipByUserID(request GetProjectMembershipByUserIDRequest) (GetProjectUserByUserNameResponse, error) {
	var projectMembershipResponse GetProjectUserByUserNameResponse

	response, err := client.Config.HttpClient.
		R().
		SetResult(&projectMembershipResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Get(fmt.Sprintf("api/v1/projects/%s/user/%s/membership", request.ProjectID, request.UserID))

	if err != nil {
		return GetProjectUserByUserNameResponse{}, errors.NewGenericRequestError(operationGetProjectMembershipByUserID, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return GetProjectUserByUserNameResponse{}, ErrNotFound
		}
		return GetProjectUserByUserNameResponse{}, errors.NewAPIErrorWithResponse(operationGetProjectMembershipByUserID, response, nil)
	}

	return projectMembershipResponse, nil
}
