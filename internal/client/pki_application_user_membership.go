package infisicalclient

import (
	"fmt"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationAddPkiApplicationUserMembers       = "CallAddPkiApplicationUserMembers"
	operationListPkiApplicationUserMembers      = "CallListPkiApplicationUserMembers"
	operationUpdatePkiApplicationUserMemberRole = "CallUpdatePkiApplicationUserMemberRole"
	operationRemovePkiApplicationUserMember     = "CallRemovePkiApplicationUserMember"
)

func (client Client) AddPkiApplicationUserMembers(request AddPkiApplicationUserMembersRequest) (AddPkiApplicationUserMembersResponse, error) {
	var membersResponse AddPkiApplicationUserMembersResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&membersResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(fmt.Sprintf("api/v1/cert-manager/applications/%s/users", request.ApplicationId))

	if err != nil {
		return AddPkiApplicationUserMembersResponse{}, errors.NewGenericRequestError(operationAddPkiApplicationUserMembers, err)
	}

	if response.IsError() {
		return AddPkiApplicationUserMembersResponse{}, errors.NewAPIErrorWithResponse(operationAddPkiApplicationUserMembers, response, nil)
	}

	return membersResponse, nil
}

func (client Client) ListPkiApplicationUserMembers(request ListPkiApplicationUserMembersRequest) (ListPkiApplicationUserMembersResponse, error) {
	var membersResponse ListPkiApplicationUserMembersResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&membersResponse).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/cert-manager/applications/%s/users", request.ApplicationId))

	if err != nil {
		return ListPkiApplicationUserMembersResponse{}, errors.NewGenericRequestError(operationListPkiApplicationUserMembers, err)
	}

	if response.IsError() {
		if response.StatusCode() == 404 || response.StatusCode() == 422 {
			return ListPkiApplicationUserMembersResponse{}, ErrNotFound
		}
		return ListPkiApplicationUserMembersResponse{}, errors.NewAPIErrorWithResponse(operationListPkiApplicationUserMembers, response, nil)
	}

	return membersResponse, nil
}

func (client Client) UpdatePkiApplicationUserMemberRole(request UpdatePkiApplicationUserMemberRoleRequest) (UpdatePkiApplicationUserMemberRoleResponse, error) {
	var memberResponse UpdatePkiApplicationUserMemberRoleResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&memberResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/cert-manager/applications/%s/users/%s", request.ApplicationId, request.UserId))

	if err != nil {
		return UpdatePkiApplicationUserMemberRoleResponse{}, errors.NewGenericRequestError(operationUpdatePkiApplicationUserMemberRole, err)
	}

	if response.IsError() {
		return UpdatePkiApplicationUserMemberRoleResponse{}, errors.NewAPIErrorWithResponse(operationUpdatePkiApplicationUserMemberRole, response, nil)
	}

	return memberResponse, nil
}

func (client Client) RemovePkiApplicationUserMember(request RemovePkiApplicationUserMemberRequest) (RemovePkiApplicationUserMemberResponse, error) {
	var memberResponse RemovePkiApplicationUserMemberResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&memberResponse).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/cert-manager/applications/%s/users/%s", request.ApplicationId, request.UserId))

	if err != nil {
		return RemovePkiApplicationUserMemberResponse{}, errors.NewGenericRequestError(operationRemovePkiApplicationUserMember, err)
	}

	if response.IsError() {
		return RemovePkiApplicationUserMemberResponse{}, errors.NewAPIErrorWithResponse(operationRemovePkiApplicationUserMember, response, nil)
	}

	return memberResponse, nil
}
