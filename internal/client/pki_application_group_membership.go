package infisicalclient

import (
	"fmt"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationListPkiApplicationGroupMembers      = "CallListPkiApplicationGroupMembers"
	operationAddPkiApplicationGroupMember        = "CallAddPkiApplicationGroupMember"
	operationUpdatePkiApplicationGroupMemberRole = "CallUpdatePkiApplicationGroupMemberRole"
	operationRemovePkiApplicationGroupMember     = "CallRemovePkiApplicationGroupMember"
)

func (client Client) ListPkiApplicationGroupMembers(request ListPkiApplicationGroupMembersRequest) (ListPkiApplicationGroupMembersResponse, error) {
	var membersResponse ListPkiApplicationGroupMembersResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&membersResponse).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/cert-manager/applications/%s/groups", request.ApplicationId))

	if err != nil {
		return ListPkiApplicationGroupMembersResponse{}, errors.NewGenericRequestError(operationListPkiApplicationGroupMembers, err)
	}

	if response.IsError() {
		if response.StatusCode() == 404 || response.StatusCode() == 422 {
			return ListPkiApplicationGroupMembersResponse{}, ErrNotFound
		}
		return ListPkiApplicationGroupMembersResponse{}, errors.NewAPIErrorWithResponse(operationListPkiApplicationGroupMembers, response, nil)
	}

	return membersResponse, nil
}

func (client Client) AddPkiApplicationGroupMember(request AddPkiApplicationGroupMemberRequest) (AddPkiApplicationGroupMemberResponse, error) {
	var memberResponse AddPkiApplicationGroupMemberResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&memberResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(fmt.Sprintf("api/v1/cert-manager/applications/%s/groups/%s", request.ApplicationId, request.GroupId))

	if err != nil {
		return AddPkiApplicationGroupMemberResponse{}, errors.NewGenericRequestError(operationAddPkiApplicationGroupMember, err)
	}

	if response.IsError() {
		return AddPkiApplicationGroupMemberResponse{}, errors.NewAPIErrorWithResponse(operationAddPkiApplicationGroupMember, response, nil)
	}

	return memberResponse, nil
}

func (client Client) UpdatePkiApplicationGroupMemberRole(request UpdatePkiApplicationGroupMemberRoleRequest) (UpdatePkiApplicationGroupMemberRoleResponse, error) {
	var memberResponse UpdatePkiApplicationGroupMemberRoleResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&memberResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/cert-manager/applications/%s/groups/%s", request.ApplicationId, request.GroupId))

	if err != nil {
		return UpdatePkiApplicationGroupMemberRoleResponse{}, errors.NewGenericRequestError(operationUpdatePkiApplicationGroupMemberRole, err)
	}

	if response.IsError() {
		return UpdatePkiApplicationGroupMemberRoleResponse{}, errors.NewAPIErrorWithResponse(operationUpdatePkiApplicationGroupMemberRole, response, nil)
	}

	return memberResponse, nil
}

func (client Client) RemovePkiApplicationGroupMember(request RemovePkiApplicationGroupMemberRequest) (RemovePkiApplicationGroupMemberResponse, error) {
	var memberResponse RemovePkiApplicationGroupMemberResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&memberResponse).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/cert-manager/applications/%s/groups/%s", request.ApplicationId, request.GroupId))

	if err != nil {
		return RemovePkiApplicationGroupMemberResponse{}, errors.NewGenericRequestError(operationRemovePkiApplicationGroupMember, err)
	}

	if response.IsError() {
		return RemovePkiApplicationGroupMemberResponse{}, errors.NewAPIErrorWithResponse(operationRemovePkiApplicationGroupMember, response, nil)
	}

	return memberResponse, nil
}
