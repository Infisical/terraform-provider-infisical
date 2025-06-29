package infisicalclient

import (
	"fmt"
	"net/http"
	"terraform-provider-infisical/internal/errors"
)

const (
	operationCreateGroup  = "CallCreateGroup"
	operationUpdateGroup  = "CallUpdateGroup"
	operationDeleteGroup  = "CallDeleteGroup"
	operationGetGroupById = "CallGetGroupById"
	operationGetGroups    = "CallGetGroups"
)

func (client Client) CreateGroup(request CreateGroupRequest) (Group, error) {
	var groupResponse Group
	response, err := client.Config.HttpClient.
		R().
		SetResult(&groupResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Post("api/v1/groups")

	if err != nil {
		return Group{}, errors.NewGenericRequestError(operationCreateGroup, err)
	}

	if response.IsError() {
		return Group{}, errors.NewAPIErrorWithResponse(operationCreateGroup, response, nil)
	}

	return groupResponse, nil
}

func (client Client) UpdateGroup(request UpdateGroupRequest) (Group, error) {
	var groupResponse Group
	response, err := client.Config.HttpClient.
		R().
		SetResult(&groupResponse).
		SetHeader("User-Agent", USER_AGENT).
		SetBody(request).
		Patch(fmt.Sprintf("api/v1/groups/%s", request.ID))

	if err != nil {
		return Group{}, errors.NewGenericRequestError(operationUpdateGroup, err)
	}

	if response.IsError() {
		return Group{}, errors.NewAPIErrorWithResponse(operationUpdateGroup, response, nil)
	}

	return groupResponse, nil
}

func (client Client) DeleteGroup(request DeleteGroupRequest) (Group, error) {
	var groupResponse Group
	response, err := client.Config.HttpClient.
		R().
		SetResult(&groupResponse).
		SetHeader("User-Agent", USER_AGENT).
		Delete(fmt.Sprintf("api/v1/groups/%s", request.ID))

	if err != nil {
		return Group{}, errors.NewGenericRequestError(operationDeleteGroup, err)
	}

	if response.IsError() {
		return Group{}, errors.NewAPIErrorWithResponse(operationDeleteGroup, response, nil)
	}

	return groupResponse, nil
}

func (client Client) GetGroupById(request GetGroupByIdRequest) (Group, error) {
	var groupResponse Group
	response, err := client.Config.HttpClient.
		R().
		SetResult(&groupResponse).
		SetHeader("User-Agent", USER_AGENT).
		Get(fmt.Sprintf("api/v1/groups/%s", request.ID))

	if response.StatusCode() == http.StatusNotFound {
		return Group{}, ErrNotFound
	}

	if err != nil {
		return Group{}, errors.NewGenericRequestError(operationGetGroupById, err)
	}

	if response.IsError() {
		return Group{}, errors.NewAPIErrorWithResponse(operationGetGroupById, response, nil)
	}

	return groupResponse, nil
}

func (client Client) GetGroups() (GetGroupsResponse, error) {
	var body GetGroupsResponse

	httpRequest := client.Config.HttpClient.
		R().
		SetResult(&body).
		SetHeader("User-Agent", USER_AGENT)

	response, err := httpRequest.Get("api/v1/groups")

	if err != nil {
		return GetGroupsResponse{}, errors.NewGenericRequestError(operationGetGroups, err)
	}

	if response.IsError() {
		return GetGroupsResponse{}, errors.NewAPIErrorWithResponse(operationGetGroups, response, nil)
	}

	return body, nil
}
