package infisicalclient

import (
	"fmt"
	"net/http"
	"strings"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationListPamMembers      = "CallListPamMembers"
	operationAddPamMember        = "CallAddPamMember"
	operationAddPamProductUsers  = "CallAddPamProductUsers"
	operationUpdatePamMemberRole = "CallUpdatePamMemberRole"
	operationRemovePamMember     = "CallRemovePamMember"
)

func pamMembersPath(scope PamMembershipScope, kind PamMemberKind) string {
	if scope.Resource == "" {
		return fmt.Sprintf("api/v1/pam/memberships/%s", kind)
	}
	return fmt.Sprintf("api/v1/pam/%s/%s/%s", scope.Resource, scope.ResourceID, kind)
}

func (client Client) ListPamMembers(request ListPamMembersRequest) (ListPamMembersResponse, error) {
	var body ListPamMembersResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		Get(pamMembersPath(request.Scope, request.Kind))

	if err != nil {
		return ListPamMembersResponse{}, errors.NewGenericRequestError(operationListPamMembers, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return ListPamMembersResponse{}, ErrNotFound
		}
		return ListPamMembersResponse{}, errors.NewAPIErrorWithResponse(operationListPamMembers, response, nil)
	}

	return body, nil
}

func (client Client) AddPamMember(request AddPamMemberRequest) (PamMember, error) {
	var body PamMember
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(fmt.Sprintf("%s/%s", pamMembersPath(request.Scope, request.Kind), request.MemberID))

	if err != nil {
		return PamMember{}, errors.NewGenericRequestError(operationAddPamMember, err)
	}

	if response.IsError() {
		return PamMember{}, errors.NewAPIErrorWithResponse(operationAddPamMember, response, nil)
	}

	return body, nil
}

func (client Client) AddPamProductUsers(request AddPamProductUsersRequest) (AddPamProductUsersResponse, error) {
	var body AddPamProductUsersResponse
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post(pamMembersPath(PamMembershipScope{}, PamMemberKindUser))

	if err != nil {
		return AddPamProductUsersResponse{}, errors.NewGenericRequestError(operationAddPamProductUsers, err)
	}

	if response.IsError() {
		return AddPamProductUsersResponse{}, errors.NewAPIErrorWithResponse(operationAddPamProductUsers, response, nil)
	}

	return body, nil
}

func (client Client) UpdatePamMemberRole(request UpdatePamMemberRoleRequest) (PamMember, error) {
	var body PamMember
	response, err := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("%s/%s", pamMembersPath(request.Scope, request.Kind), request.MemberID))

	if err != nil {
		return PamMember{}, errors.NewGenericRequestError(operationUpdatePamMemberRole, err)
	}

	if response.IsError() {
		return PamMember{}, errors.NewAPIErrorWithResponse(operationUpdatePamMemberRole, response, nil)
	}

	return body, nil
}

func (client Client) RemovePamMember(request RemovePamMemberRequest) error {
	response, err := client.Config.HttpClient.
		R().
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("%s/%s", pamMembersPath(request.Scope, request.Kind), request.MemberID))

	if err != nil {
		return errors.NewGenericRequestError(operationRemovePamMember, err)
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			return ErrNotFound
		}
		return errors.NewAPIErrorWithResponse(operationRemovePamMember, response, nil)
	}

	return nil
}

func (client Client) FindPamProductUserByEmail(email string) (string, error) {
	members, err := client.ListPamMembers(ListPamMembersRequest{Kind: PamMemberKindUser})
	if err != nil {
		return "", err
	}

	// Email matches win over username matches, and more than one match is an error rather than a guess.
	var byEmail, byUsername []string
	for _, member := range members.Members {
		if member.UserID == nil {
			continue
		}
		if member.Email != nil && strings.EqualFold(*member.Email, email) {
			byEmail = append(byEmail, *member.UserID)
		} else if member.Username != nil && strings.EqualFold(*member.Username, email) {
			byUsername = append(byUsername, *member.UserID)
		}
	}

	for _, matches := range [][]string{byEmail, byUsername} {
		if len(matches) == 1 {
			return matches[0], nil
		}
		if len(matches) > 1 {
			return "", fmt.Errorf("more than one PAM member matches %q", email)
		}
	}

	return "", ErrNotFound
}
