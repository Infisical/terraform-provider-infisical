package infisicalclient

import (
	"fmt"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationListPkiApplicationIdentityMembers      = "CallListPkiApplicationIdentityMembers"
	operationAddPkiApplicationIdentityMember        = "CallAddPkiApplicationIdentityMember"
	operationUpdatePkiApplicationIdentityMemberRole = "CallUpdatePkiApplicationIdentityMemberRole"
	operationRemovePkiApplicationIdentityMember     = "CallRemovePkiApplicationIdentityMember"
)

func (client Client) ListPkiApplicationIdentityMembers(request ListPkiApplicationIdentityMembersRequest) (ListPkiApplicationIdentityMembersResponse, error) {
	var membersResponse ListPkiApplicationIdentityMembersResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&membersResponse).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/cert-manager/applications/%s/identities", request.ApplicationId))

	if err != nil {
		return ListPkiApplicationIdentityMembersResponse{}, errors.NewGenericRequestError(operationListPkiApplicationIdentityMembers, err)
	}

	if response.IsError() {
		if response.StatusCode() == 404 || response.StatusCode() == 422 {
			return ListPkiApplicationIdentityMembersResponse{}, ErrNotFound
		}
		return ListPkiApplicationIdentityMembersResponse{}, errors.NewAPIErrorWithResponse(operationListPkiApplicationIdentityMembers, response, nil)
	}

	return membersResponse, nil
}

func (client Client) AddPkiApplicationIdentityMember(request AddPkiApplicationIdentityMemberRequest) (AddPkiApplicationIdentityMemberResponse, error) {
	var memberResponse AddPkiApplicationIdentityMemberResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&memberResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(fmt.Sprintf("api/v1/cert-manager/applications/%s/identities/%s", request.ApplicationId, request.IdentityId))

	if err != nil {
		return AddPkiApplicationIdentityMemberResponse{}, errors.NewGenericRequestError(operationAddPkiApplicationIdentityMember, err)
	}

	if response.IsError() {
		return AddPkiApplicationIdentityMemberResponse{}, errors.NewAPIErrorWithResponse(operationAddPkiApplicationIdentityMember, response, nil)
	}

	return memberResponse, nil
}

func (client Client) UpdatePkiApplicationIdentityMemberRole(request UpdatePkiApplicationIdentityMemberRoleRequest) (UpdatePkiApplicationIdentityMemberRoleResponse, error) {
	var memberResponse UpdatePkiApplicationIdentityMemberRoleResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&memberResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/cert-manager/applications/%s/identities/%s", request.ApplicationId, request.IdentityId))

	if err != nil {
		return UpdatePkiApplicationIdentityMemberRoleResponse{}, errors.NewGenericRequestError(operationUpdatePkiApplicationIdentityMemberRole, err)
	}

	if response.IsError() {
		return UpdatePkiApplicationIdentityMemberRoleResponse{}, errors.NewAPIErrorWithResponse(operationUpdatePkiApplicationIdentityMemberRole, response, nil)
	}

	return memberResponse, nil
}

func (client Client) RemovePkiApplicationIdentityMember(request RemovePkiApplicationIdentityMemberRequest) (RemovePkiApplicationIdentityMemberResponse, error) {
	var memberResponse RemovePkiApplicationIdentityMemberResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&memberResponse).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/cert-manager/applications/%s/identities/%s", request.ApplicationId, request.IdentityId))

	if err != nil {
		return RemovePkiApplicationIdentityMemberResponse{}, errors.NewGenericRequestError(operationRemovePkiApplicationIdentityMember, err)
	}

	if response.IsError() {
		return RemovePkiApplicationIdentityMemberResponse{}, errors.NewAPIErrorWithResponse(operationRemovePkiApplicationIdentityMember, response, nil)
	}

	return memberResponse, nil
}
